package fzapi

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jhoonb/archivex"
	"golang.org/x/crypto/pkcs12"
)

// FileZenRAUserAgent : User-Agentは、子機と同じ、FileZenの設定で制御可能
const FileZenRAUserAgent = "FileZenRA"

// gBody : 分割アップロード用のバッファ
var gBody = &bytes.Buffer{}

// XMLFile : FileZenの応答内のファイルを表すstruct
type XMLFile struct {
	XMLName   xml.Name `xml:"File"`
	DrmFlag   string   `xml:"DrmFlag,attr"`
	Key       string   `xml:"Key,attr"`
	Name      string   `xml:"Name,attr"`
	Owner     string   `xml:"Owner,attr"`
	PdfFlag   string   `xml:"PdfFlag,attr"`
	Size      string   `xml:"Size,attr"`
	TimeStamp string   `xml:"TimeStamp,attr"`
}

// XMLFolder : FileZenの応答内のフォルダを表すstruct
type XMLFolder struct {
	XMLName  xml.Name   `xml:"Folder"`
	Name     string     `xml:"Name,attr"`
	Access   string     `xml:"Access,attr"`
	ID       string     `xml:"Id,attr"`
	Limit    string     `xml:"Limit,attr"`
	FileList []*XMLFile `xml:"File"`
}

// XMLProject : FileZenの応答内のプロジェクトを表すstruct
type XMLProject struct {
	XMLName    xml.Name     `xml:"Project"`
	Name       string       `xml:"Name,attr"`
	FolderList []*XMLFolder `xml:"Folder"`
}

// XMLFileZen : FileZenの応答を表すstruct
type XMLFileZen struct {
	XMLName        xml.Name      `xml:"FileZen"`
	Res            string        `xml:"Lastop>Res"`
	ProjectList    []*XMLProject `xml:"ProjectList>Project"`
	SystemMailAddr string        `xml:"SystemMailAddr"`
	UserMailAddr   string        `xml:"UserMailAddr"`
	ValidKey       string        `xml:"ValidKey"`
	Version        string        `xml:"Version"`
}

// FzAPI : FileZen APIを表すstruct
type FzAPI struct {
	URL                string
	InsecureSkipVerify bool
	Timeout            int
	CaCert             []byte
	UseClientCert      bool
	ClientCert         tls.Certificate
	FzSession          http.Cookie
	LastResp           *XMLFileZen
	client             *http.Client
}

// getHTTPClient : TLSの設定などを使って、ＨＴＴＰクライアントを作成する
func (fz *FzAPI) getHTTPClient() {
	tlsConfig := &tls.Config{InsecureSkipVerify: fz.InsecureSkipVerify}
	if len(fz.CaCert) > 0 {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(fz.CaCert)
		tlsConfig.RootCAs = caCertPool
	}
	if fz.UseClientCert {
		tlsConfig.Certificates = []tls.Certificate{fz.ClientCert}
	}
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	fz.client = &http.Client{Timeout: time.Duration(fz.Timeout) * time.Second, Transport: tr}
}

// ParseXMLResp : FileZenの応答解析
func (fz *FzAPI) ParseXMLResp(r *http.Response) error {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("ParseXMLResp - ReadAll Error: %v", err)
	}
	fzResp := &XMLFileZen{}
	err = xml.Unmarshal(body, fzResp)
	if err != nil {
		return fmt.Errorf("ParseXMLResp - Unmarshal Error: %v", err)
	}
	if fzResp.Res == "OK" {
		fz.LastResp = fzResp
		return nil
	}
	return errors.New(fzResp.Res)
}

// getSessionID : セッションIDを取り出す
func (fz *FzAPI) getSessionID(r *http.Response) {
	for _, c := range r.Cookies() {
		if c.Name == "SessionID" {
			fz.FzSession = *c
			return
		}
	}
	fz.FzSession = http.Cookie{Name: "SessionID", Value: ""}
}

// FzSendPostReq : FileZenはPOSTリクエストを送信する
func (fz *FzAPI) FzSendPostReq(url string, body io.Reader, bSetCookie bool) (*http.Response, error) {
	if fz.client == nil {
		fz.getHTTPClient()
	}
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("User-Agent", FileZenRAUserAgent)
	if bSetCookie {
		req.AddCookie(&fz.FzSession)
	}
	return fz.client.Do(req)
}

// FzLogin : FileZenへログインする
func (fz *FzAPI) FzLogin(fzURL, uid, password string) error {
	fz.URL = fzURL
	v := url.Values{}
	v.Set("respmode", "xml")
	v.Set("action", "Login")
	v.Set("sub_action", "auth")
	v.Set("user_id", uid)
	v.Set("password", password)
	resp, err := fz.FzSendPostReq(fz.URL+"/cgi-bin/index.cgi", strings.NewReader(v.Encode()), false)
	if err != nil {
		return fmt.Errorf("FzLogin - FzSendPostReq Error: %v", err)
	}
	err = fz.ParseXMLResp(resp)
	if err != nil {
		return fmt.Errorf("FzLogin - ParseXmlResp Error: %v", err)
	}
	fz.getSessionID(resp)
	return nil
}

// FzLogout : FileZenからログアウトする
func (fz *FzAPI) FzLogout() error {
	v := url.Values{}
	v.Set("respmode", "xml")
	v.Set("action", "Logout")
	v.Set("sub_action", "show")
	_, err := fz.FzSendPostReq(fz.URL+"/cgi-bin/index.cgi", strings.NewReader(v.Encode()), true)
	if err != nil {
		return fmt.Errorf("FzLogout - FzSendPostReq Error: %v", err)
	}
	return nil
}

// FzReload : FielZenのログイン情報を更新する（ファイルやフォルダなどのリストの更新）
func (fz *FzAPI) FzReload() error {
	v := url.Values{}
	v.Set("respmode", "xml")
	v.Set("action", "Mainmenu_file")
	v.Set("sub_action", "show")
	v.Set("valid_key", fz.LastResp.ValidKey)
	resp, err := fz.FzSendPostReq(fz.URL+"/cgi-bin/index.cgi", strings.NewReader(v.Encode()), true)
	if err != nil {
		return fmt.Errorf("FzReload - FzSendPostReq Error: %v", err)
	}
	return fz.ParseXMLResp(resp)
}

// FzDeleteFile : FileZenのプロジェクト上のファイルを削除する
func (fz *FzAPI) FzDeleteFile(key string) error {
	v := url.Values{}
	v.Set("respmode", "xml")
	v.Set("action", "Mainmenu_file")
	v.Set("sub_action", "delete_file")
	v.Set("key", key)
	v.Set("valid_key", fz.LastResp.ValidKey)
	resp, err := fz.FzSendPostReq(fz.URL+"/cgi-bin/index.cgi", strings.NewReader(v.Encode()), true)
	if err != nil {
		return fmt.Errorf("FzDeleteFile - FzSendPostReq Error: %v", err)
	}
	return fz.ParseXMLResp(resp)
}

// FzFindFile : FileZenの指定のフォルダ（プロジェクト）内のファイルを探す
func (fz *FzAPI) FzFindFile(prj string, folder string, file string) string {
	for _, p := range fz.LastResp.ProjectList {
		if p.Name == prj {
			for _, d := range p.FolderList {
				if d.Name == folder {
					for _, f := range d.FileList {
						if f.Name == file {
							return f.Key
						}
					}
				}
			}
		}
	}
	return ""
}

// sepPrjFolder : プロジェクトとフォルダ名を分離する
func sepPrjFolder(prjFolder string) (string, string) {
	a := strings.Split(prjFolder, "/")
	if len(a) < 2 {
		return prjFolder, ""
	}
	return a[0], a[1]
}

// FzFindFolder : FileZenのフォルダを名前から探す
func (fz *FzAPI) FzFindFolder(prjFolder string) *XMLFolder {
	prj, folder := sepPrjFolder(prjFolder)
	for _, p := range fz.LastResp.ProjectList {
		if p.Name == prj {
			for _, d := range p.FolderList {
				if d.Name == folder {
					return d
				}
			}
		}
	}
	return &XMLFolder{}
}

// CanUpload : FileZenへファイルをアップロード可能かどうか判断する
func (fz *FzAPI) CanUpload(prjFolder string, file string) bool {
	prj, folder := sepPrjFolder(prjFolder)
	ret := false
	for _, p := range fz.LastResp.ProjectList {
		if p.Name == prj {
			for _, d := range p.FolderList {
				if d.Name == folder {
					for _, f := range d.FileList {
						if f.Name == file {
							return false
						}
					}
					// アクセス可能なFolderがある場合
					if strings.Index(d.Access, "write") != -1 {
						ret = true
					}
				}
			}
		}
	}
	return ret
}

// FzDownload : FileZenからファイルをダウンロードする
func (fz *FzAPI) FzDownload(key string, localFile string) error {
	v := url.Values{}
	v.Set("respmode", "xml")
	v.Set("action", "Mainmenu_file")
	v.Set("sub_action", "download")
	v.Set("key", key)
	v.Set("valid_key", fz.LastResp.ValidKey)
	resp, err := fz.FzSendPostReq(fz.URL+"/cgi-bin/index.cgi", strings.NewReader(v.Encode()), true)
	if err != nil {
		return fmt.Errorf("FzDownload - FzSendPostReq Error: %v", err)
	}
	defer resp.Body.Close()
	if resp.ContentLength < 0 {
		err = fz.ParseXMLResp(resp)
		if err != nil {
			return fmt.Errorf("FzDownload - ParseXmlResp Error: %v", err)
		}
	}
	output, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("FzDownload - os.Create Error: %v", err)
	}
	defer output.Close()
	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("FzDownload - io.Copy Error: %v", err)
	}
	return nil
}

// secureRandam : 分割アップロードに使用する乱数キーを作成する
func secureRandam(c int) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d", rand.Intn(999999999))
}

// fzPlUploadPart : FileZenへリクエストを分割してファイルをアップロードする場合の
// １ファイルのアップロード処理
func (fz *FzAPI) fzPlUploadPart(src io.Reader, nSize int64, fr, folderID string, chunk, chunks int) (bool, error) {
	writer := multipart.NewWriter(gBody)
	part, err := writer.CreateFormFile("file", fr+folderID+".tmp")
	if err != nil {
		return false, fmt.Errorf("FzPlUploadPart - CreateFormFile Error: %v", err)
	}
	_, err = io.CopyN(part, src, nSize)
	_ = writer.WriteField("fr", fr)
	_ = writer.WriteField("valid_key", fz.LastResp.ValidKey)
	_ = writer.WriteField("action", "Mainmenu_upload")
	_ = writer.WriteField("sub_action", "plupload")
	_ = writer.WriteField("ukey", fz.FzSession.Value+folderID)
	_ = writer.WriteField("mode", "PRJ")
	_ = writer.WriteField("chunk", strconv.Itoa(chunk))
	_ = writer.WriteField("chunks", strconv.Itoa(chunks))
	err = writer.Close()
	if err != nil {
		return false, fmt.Errorf("FzPlUploadPart - writer.Close Error: %v", err)
	}
	req, _ := http.NewRequest("POST", fz.URL+"/cgi-bin/index.cgi", gBody)
	req.Header.Set("User-Agent", FileZenRAUserAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&fz.FzSession)
	resp, err := fz.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("FzPlUploadPart - POST Error: %v", err)
	}
	err = resp.Body.Close()
	if err != nil {
		return false, fmt.Errorf("FzPlUploadPart - resp.Body.Close Error: %v", err)
	}
	return true, nil
}

// FzPlUpload : FileZenへファイルを分割したリクエストでアップロードする
func (fz *FzAPI) FzPlUpload(localFile, folderID, regName, comment, notifyTo, notifyMode string) error {
	fstat, err := os.Stat(localFile)
	if err != nil {
		return fmt.Errorf("FzPlUpload - os.stat Error: %v", err)
	}
	file, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("FzPlUpload - os.Open Error: %v", err)
	}
	defer file.Close()
	fr := secureRandam(10)
	nTotalSize := fstat.Size()
	var n int64
	var nSend int64
	chunk := 1
	nChunkSize := int64(1024 * 1024 * 50) // 50MB
	chunks := int(nTotalSize / nChunkSize)
	if nTotalSize%nChunkSize != 0 {
		chunks++
	}
	for n = 0; n < nTotalSize; {
		nSend = nChunkSize
		if nTotalSize-n < nChunkSize {
			nSend = nTotalSize - n
		}
		_, err = fz.fzPlUploadPart(file, nSend, fr, folderID, chunk, chunks)
		if err != nil {
			return err
		}
		n += nSend
		chunk++
	}
	v := url.Values{}
	v.Set("respmode", "xml")
	v.Set("action", "Mainmenu_upload")
	v.Set("sub_action", "do_upload")
	v.Set("valid_key", fz.LastResp.ValidKey)
	v.Set("filename", filepath.Base(localFile))
	v.Set("ST_current_folder", folderID)
	v.Set("reg_filename", regName)
	v.Set("description", comment)
	v.Set("fr", fr)
	v.Set("key", "")
	if strings.Index(notifyTo, "ALL") != -1 {
		v.Set("mail_send", "1")
	} else if notifyTo == "" {
		v.Set("mail_send", "0")
	} else {
		v.Set("mail_send", "2")
	}
	if strings.Index(notifyMode, "DOWNLOAD") != -1 {
		v.Set("notify_download", "1")
	} else {
		v.Set("notify_download", "0")
	}
	if strings.Index(notifyMode, "ALTER") != -1 {
		v.Set("notify_alter", "1")
	} else {
		v.Set("notify_alter", "0")
	}
	if strings.Index(notifyMode, "DELETE") != -1 {
		v.Set("notify_delete", "1")
	} else {
		v.Set("notify_delete", "0")
	}
	v.Set("new_alert", "1")
	resp, err := fz.FzSendPostReq(fz.URL+"/cgi-bin/index.cgi", strings.NewReader(v.Encode()), true)
	if err != nil {
		return fmt.Errorf("FzPlUpload - FzSendPostReq Error: %v", err)
	}
	return fz.ParseXMLResp(resp)
}

// FzUpload : FileZenへ１つのリクエストでファイルをアップロードする(HTMLモードと同じ)
func (fz *FzAPI) FzUpload(localFile, folderID, regName, comment, notifyTo, notifyMode string) error {
	file, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("FzUpload - os.Open Error: %v", err)
	}
	defer file.Close()
	writer := multipart.NewWriter(gBody)
	part, err := writer.CreateFormFile("filename", filepath.Base(localFile))
	if err != nil {
		return fmt.Errorf("FzUpload - writer.CreateFormFile Error: %v", err)
	}
	_, err = io.Copy(part, file)
	_ = writer.WriteField("action", "Mainmenu_upload")
	_ = writer.WriteField("sub_action", "do_upload")
	_ = writer.WriteField("respmode", "xml")
	_ = writer.WriteField("valid_key", fz.LastResp.ValidKey)
	_ = writer.WriteField("ST_current_folder", folderID)
	_ = writer.WriteField("reg_filename", regName)
	_ = writer.WriteField("description", comment)
	_ = writer.WriteField("key", "")
	if strings.Index(notifyTo, "ALL") != -1 {
		_ = writer.WriteField("mail_send", "1")
	} else if notifyTo == "" {
		_ = writer.WriteField("mail_send", "0")
	} else {
		_ = writer.WriteField("mail_send", "2")
	}
	if strings.Index(notifyMode, "DOWNLOAD") != -1 {
		_ = writer.WriteField("notify_download", "1")
	} else {
		_ = writer.WriteField("notify_download", "0")
	}
	if strings.Index(notifyMode, "ALTER") != -1 {
		_ = writer.WriteField("notify_alter", "1")
	} else {
		_ = writer.WriteField("notify_alter", "0")
	}
	if strings.Index(notifyMode, "DELETE") != -1 {
		_ = writer.WriteField("notify_delete", "1")
	} else {
		_ = writer.WriteField("notify_delete", "0")
	}
	_ = writer.WriteField("new_alert", "1")
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("FzUpload - writer.Close Error: %v", err)
	}
	req, _ := http.NewRequest("POST", fz.URL+"/cgi-bin/index.cgi", gBody)
	req.Header.Set("User-Agent", FileZenRAUserAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&fz.FzSession)
	resp, err := fz.client.Do(req)
	if err != nil {
		return fmt.Errorf("FzUpload - POST Error: %v", err)
	}
	return fz.ParseXMLResp(resp)
}

// FzGetFileComment : ファイルのコメントを作成する
func FzGetFileComment(f string) string {
	fstat, err := os.Stat(f)
	if err != nil {
		return ""
	}
	ret := ""
	ret += fmt.Sprintf("OrgPath: %s\n", f)
	ret += fmt.Sprintf("Size: %d\n", fstat.Size())
	h := sha1.New()
	fi, err := os.Open(f)
	if err != nil {
		return ret
	}
	defer fi.Close()
	buf := make([]byte, 8192)
	for {
		n, err := fi.Read(buf)
		if err != nil && err != io.EOF {
			return ret
		}
		if n == 0 {
			break
		}
		h.Write(buf[:n])
	}
	bs := h.Sum(nil)
	ret += fmt.Sprintf("SHA1: %x\n", bs)
	return ret
}

// eMailAddr: メールアドレスの内部表現
type eMailAddr struct {
	Name  string
	Email string
}

// getMbOpt: めるあど便のオプションを取得する
func getMbOpt(mb map[string]string, k string, d string) string {
	v, ok := mb[k]
	if ok {
		return v
	}
	return d
}

// getEMailAddr: めるあど便の設定ファイルからメールアドレスを取得する
func getEMailAddr(s string) eMailAddr {
	name := ""
	email := ""
	reg := regexp.MustCompile(`\s*([^<]+)\s*<\s*([^ >@]+@[^ >@]+)\s*>`)
	a := reg.FindStringSubmatch(s)
	if len(a) > 2 {
		name = a[1]
		email = a[2]
	} else {
		reg = regexp.MustCompile(`([^ @]+@[^ @]+)`)
		a = reg.FindStringSubmatch(s)
		if len(a) > 1 {
			name = a[1]
			email = a[1]
		}
	}
	ret := eMailAddr{Name: name, Email: email}
	return ret
}

/*
	loadMbConf : めるあど便の送信ファイルを読み込む
   subject: 件名
   mailto: 宛先<email>(複数指定可能)
   from: 送信元<email>
   file: ファイル名｜ディレクトリ名
   start:公開開始日
   days:公開期間
   limit: ダウンロード回数制限
   comment: このキー以降は、本文にする
*/
func loadMbConf(mbConf string) (map[string]string, error) {
	ret := map[string]string{}
	fp, err := os.Open(mbConf)
	if err != nil {
		return ret, err
	}
	defer fp.Close()
	scanner := bufio.NewScanner(fp)
	bInComment := false
	comment := ""
	err = nil
	line := ""
	for scanner.Scan() {
		if err = scanner.Err(); err != nil {
			break
		}
		line = scanner.Text()
		if bInComment {
			comment += line + "\n"
			continue
		}
		a := strings.Split(line, ":")
		if len(a) < 2 {
			continue
		}
		k := strings.ToLower(a[0])
		v := strings.TrimSpace(a[1])
		if k == "comment" {
			bInComment = true
			comment = v + "\n"
		} else if k == "mailto" {
			_, ok := ret["mailto"]
			if ok {
				ret["mailto"] += ","
				ret["mailto"] += v
			} else {
				ret["mailto"] = v
			}
		} else {
			ret[k] = v
		}
	}
	ret["comment"] = comment
	return ret, nil
}

// makeZip : ZIPファイルの作成
func makeZip(f string) (string, error) {
	zipFile, err := filepath.Abs(f)
	if err != nil {
		return "", err
	}
	zip := &archivex.ZipFile{}
	if err := zip.Create(zipFile); err != nil {
		return "", err
	}
	defer zip.Close()
	if err := zip.AddAll(f, true); err != nil {
		return "", err
	}
	return zipFile, nil
}

// FzMail : めるあど便の送信
func (fz *FzAPI) FzMail(mbConf string) error {
	mb, err := loadMbConf(mbConf)
	if err != nil {
		return fmt.Errorf("FzMail loadMbConf err=%v", err)
	}
	f, ok := mb["file"]
	if !ok {
		return fmt.Errorf("FzMail file not specified")
	}
	fstat, err := os.Stat(f)
	if err != nil {
		return fmt.Errorf("FzMail file stat err=%v", err)
	}
	if fstat.IsDir() {
		mb["file"], err = makeZip(f)
		if err != nil {
			return fmt.Errorf("FzMail makeZip err=%v", err)
		}
		defer os.Remove(mb["file"])
	}
	return fz.FzSendMB(mb)
}

// FzSendMB : めるあど便を送信する
func (fz *FzAPI) FzSendMB(mbconf map[string]string) error {
	from := getMbOpt(mbconf, "from", "")
	if from == "" {
		return errors.New("No from")
	}
	var toList []eMailAddr
	for _, to := range strings.Split(getMbOpt(mbconf, "mailto", ""), ",") {
		aTo := getEMailAddr(to)
		if aTo.Name != "" {
			toList = append(toList, aTo)
		}
	}
	if len(toList) < 1 {
		return errors.New("No mailto")
	}
	localFile := getMbOpt(mbconf, "file", "")
	if localFile == "" {
		return errors.New("No file")
	}
	st, err := time.Parse("2006/01/02", getMbOpt(mbconf, "start", ""))
	if err != nil {
		return errors.New("Invalid start date")
	}
	days := getMbOpt(mbconf, "days", "HEHEH")
	nDays, err := strconv.Atoi(days)
	if err != nil {
		return errors.New("Invalid days")
	}
	limit := getMbOpt(mbconf, "limit", "HEHEH")
	nLimitCount, err := strconv.Atoi(limit)
	if err != nil {
		return errors.New("Invalid limit")
	}
	comment := getMbOpt(mbconf, "comment", "")
	subject := getMbOpt(mbconf, "subject", "")
	password := getMbOpt(mbconf, "password", "")
	bNotifyDownload := false
	if getMbOpt(mbconf, "notify", "false") == "true" {
		bNotifyDownload = true
	}
	fstat, err := os.Stat(localFile)
	if err != nil {
		return fmt.Errorf("FzSendMB - os.Stat Error: %v", err)
	}
	if fstat.Size() > 1024*1024*200 {
		return fmt.Errorf("FzSendMB - File Size over 200MB")
	}
	file, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("FzSendMB - os.Open Error: %v", err)
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file1", filepath.Base(localFile))
	if err != nil {
		return fmt.Errorf("FzSendMB - writer.CreateFormFile Error: %v", err)
	}
	_, err = io.Copy(part, file)
	_ = writer.WriteField("subject", subject)
	_ = writer.WriteField("comment", comment)
	_ = writer.WriteField("exp_term_start_year", fmt.Sprintf("%d", st.Year()))
	_ = writer.WriteField("exp_term_start_month", fmt.Sprintf("%d", st.Month()))
	_ = writer.WriteField("exp_term_start_day", fmt.Sprintf("%d", st.Day()))
	_ = writer.WriteField("exp_term_start_hour", "00")
	_ = writer.WriteField("exp_term_start_minute", "00")
	_ = writer.WriteField("exp_term_type", "by_dur")
	_ = writer.WriteField("exp_term_duration", fmt.Sprintf("%d", nDays))
	_ = writer.WriteField("download_times", fmt.Sprintf("%d", nLimitCount))
	_ = writer.WriteField("recipients-max-id", fmt.Sprintf("%d", len(toList)))
	for i, toEnt := range toList {
		_ = writer.WriteField(fmt.Sprintf("recipient-name-%d", i+1), toEnt.Name)
		_ = writer.WriteField(fmt.Sprintf("recipient-email-%d", i+1), toEnt.Email)
	}
	_ = writer.WriteField("from_addr", "user")
	_ = writer.WriteField("from_addr_val", from)
	_ = writer.WriteField("lang", "ambi")
	_ = writer.WriteField("password", password)
	_ = writer.WriteField("password_retype", password)
	_ = writer.WriteField("file2", "")
	_ = writer.WriteField("file3", "")
	_ = writer.WriteField("file4", "")
	_ = writer.WriteField("file5", "")
	_ = writer.WriteField("respmode", "xml")
	_ = writer.WriteField("valid_key", fz.LastResp.ValidKey)
	_ = writer.WriteField("key", "")
	if bNotifyDownload {
		_ = writer.WriteField("notify_download", "1")
	} else {
		_ = writer.WriteField("notify_download", "0")
	}
	_ = writer.WriteField("new_alert", "1")
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("FzSendMB - writer.Close Error: %v", err)
	}
	req, _ := http.NewRequest("POST", fz.URL+"/mb/cgi-bin/index.cgi/job/api_send/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", FileZenRAUserAgent)
	req.AddCookie(&fz.FzSession)
	resp, err := fz.client.Do(req)
	if err != nil {
		return fmt.Errorf("FzSendMB - POST Error: %v", err)
	}
	return fz.ParseXMLResp(resp)
}

// FzExportCSV : FileZenから指定したCSV設定ファイルをダウンロードする
func (fz *FzAPI) FzExportCSV(params map[string]string, localFile string) error {
	output, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("FzExportCSV - os.Create Error: %v", err)
	}
	defer output.Close()
	v := url.Values{}
	v.Set("respmode", "csv")
	for key, val := range params {
		v.Set(key, val)
	}
	v.Set("valid_key", fz.LastResp.ValidKey)
	resp, err := fz.FzSendPostReq(fz.URL+"/cgi-bin/index.cgi", strings.NewReader(v.Encode()), true)
	if err != nil {
		return fmt.Errorf("FzExportCSV - FzSendPostReq Error: %v", err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("FzExportCSV - io.Copy Error: %v", err)
	}
	return nil
}

// FzImportCSV : FileZenへCSV設定ファイルをアップロードする（インポート）
func (fz *FzAPI) FzImportCSV(params map[string]string, localFile, fileKey string) error {
	file, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("FzImportCSV - os.Open Error: %v", err)
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fileKey, filepath.Base(localFile))
	if err != nil {
		return fmt.Errorf("FzImportCSV - writer.CreateFormFile Error: %v", err)
	}
	_, err = io.Copy(part, file)
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	_ = writer.WriteField("respmode", "xml")
	_ = writer.WriteField("valid_key", fz.LastResp.ValidKey)
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("FzImportCSV - writer.Close Error: %v", err)
	}
	req, _ := http.NewRequest("POST", fz.URL+"/cgi-bin/index.cgi", body)
	req.Header.Set("User-Agent", FileZenRAUserAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&fz.FzSession)
	resp, err := fz.client.Do(req)
	if err != nil {
		return fmt.Errorf("FzImportCSV - Post Error: %v", err)
	}
	return fz.ParseXMLResp(resp)
}

// MbLogExport : めるあど便の履歴をダウンロードする
func (fz *FzAPI) MbLogExport(params map[string]string, localFile string) error {
	output, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("MbLogExport - os.Create Error: %v", err)
	}
	defer output.Close()
	v := url.Values{}
	v.Set("respmode", "csv")
	for key, val := range params {
		v.Set(key, val)
	}
	v.Set("valid_key", fz.LastResp.ValidKey)
	req, _ := http.NewRequest("POST", fz.URL+"/mb/cgi-bin/index.cgi/admin/history/", strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", FileZenRAUserAgent)
	req.AddCookie(&fz.FzSession)
	resp, err := fz.client.Do(req)
	if err != nil {
		return fmt.Errorf("MbLogExport - POST Error: %v", err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("MbLogExport - io.Copy Error: %v", err)
	}
	return nil
}

// MbExportCSV : めるあど便関連のCSVエクスポート
func (fz *FzAPI) MbExportCSV(path, localFile string) error {
	output, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("MbCsvExport - os.Create Error: %v", err)
	}
	defer output.Close()
	v := url.Values{}
	req, _ := http.NewRequest("GET", fz.URL+"/mb/cgi-bin/index.cgi"+path, strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", FileZenRAUserAgent)
	req.Header.Set("Accept-Language", "ja,en")
	req.AddCookie(&fz.FzSession)
	resp, err := fz.client.Do(req)
	if err != nil {
		return fmt.Errorf("MbCsvExport - POST Error: %v", err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("MbCsvExport - io.Copy Error: %v", err)
	}
	return nil
}

// ErrRedirectAttempted : 応答がリダイレクトになる場合のエラー定義
var ErrRedirectAttempted = errors.New("redirect")

// MbImportCSV : めるあど便関連のCSVインポート
func (fz *FzAPI) MbImportCSV(path, uid, localFile, fileKey string, replace bool) error {
	file, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("MbImportCSV - os.Open Error: %v", err)
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fileKey, filepath.Base(localFile))
	if err != nil {
		return fmt.Errorf("MbImportCSV - writer.CreateFormFile Error: %v", err)
	}
	_, _ = io.Copy(part, file)
	_ = writer.WriteField("action", "import")
	if uid != "" {
		_ = writer.WriteField("uid", uid)
	}
	if replace {
		_ = writer.WriteField("replace", "yes")
	}
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("MbImportCSV - writer.Close Error: %v", err)
	}
	req, _ := http.NewRequest("POST", fz.URL+"/mb/cgi-bin/index.cgi"+path, body)
	req.Header.Set("User-Agent", FileZenRAUserAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept-Language", "ja,en")
	req.AddCookie(&fz.FzSession)
	fz.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return ErrRedirectAttempted
	}
	defer func() {
		fz.client.CheckRedirect = nil
	}()
	resp, err := fz.client.Do(req)
	if err != nil && resp != nil && resp.StatusCode != 302 {
		return fmt.Errorf("MbImportCSV - Post Error: %v", err)
	}
	loc := resp.Header.Get("Location")
	if strings.Contains(loc, "/import") {
		return fmt.Errorf("MbImportCSV - redirect error: %s", loc)
	}
	bp, err := ioutil.ReadAll(resp.Body)
	if err == nil && strings.Contains(string(bp), "アドレス帳ファイルが正しくありません") {
		return fmt.Errorf("MbImportCSV - Invalid format")
	}
	if err == nil && strings.Contains(string(bp), "ユーザーIDの指定が正しくありません。") {
		return fmt.Errorf("MbImportCSV - Invalid uid")
	}
	return nil
}

// ErrInvalidClientCert : 不正なクライアント証明書形式
var ErrInvalidClientCert = fmt.Errorf("Invalid Client Cert")

// ImportClientCert : クラアント証明書をFileZenで利用できる形式に変換する
func ImportClientCert(infile, inpass, outfile, outpass string) error {
	impcert, err := ioutil.ReadFile(infile)
	if err != nil {
		return err
	}
	of, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer of.Close()
	var pb1, pb2 *pem.Block
	pb1, impcert = pem.Decode(impcert)
	if pb1 == nil {
		pb1, pb2 = tryImportPkcs12(impcert, inpass)
		if pb1 == nil {
			return ErrInvalidClientCert
		}
	} else {
		if len(impcert) < 1 {
			return ErrInvalidClientCert
		}
		pb2, _ = pem.Decode(impcert)
	}
	if pb1.Type == "RSA PRIVATE KEY" {
		pb1, pb2 = pb2, pb1
	}
	kpem, err := getByteFromPemBlock(pb2, inpass)
	if err != nil {
		return err
	}
	blockType := "RSA PRIVATE KEY"
	password := []byte(outpass)
	cipherType := x509.PEMCipherAES256
	epb, err := x509.EncryptPEMBlock(crand.Reader,
		blockType,
		kpem,
		password,
		cipherType)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(of)
	err = pem.Encode(w, pb1)
	if err != nil {
		return err
	}
	err = pem.Encode(w, epb)
	if err != nil {
		return err
	}
	w.Flush()
	return nil
}

func tryImportPkcs12(p12 []byte, inpasswd string) (*pem.Block, *pem.Block) {
	blocks, err := pkcs12.ToPEM(p12, inpasswd)
	if err != nil || len(blocks) != 2 {
		return nil, nil
	}
	return blocks[0], blocks[1]
}

func getByteFromPemBlock(pb *pem.Block, keypass string) ([]byte, error) {
	if !x509.IsEncryptedPEMBlock(pb) {
		return pb.Bytes, nil
	}
	b, err := x509.DecryptPEMBlock(pb, []byte(keypass))
	if err == nil {
		return b, nil
	}
	return pb.Bytes, err
}

// LoadClientCert : FlieZenのクライアント証明書を読み込む
func (fz *FzAPI) LoadClientCert(cert, keypass string) error {
	clcert, err := ioutil.ReadFile(cert)
	if err != nil {
		return err
	}
	pb1, clcert := pem.Decode(clcert)
	if len(clcert) < 1 || pb1 == nil {
		return ErrInvalidClientCert
	}
	pb2, _ := pem.Decode(clcert)
	if pb2 == nil {
		return ErrInvalidClientCert
	}
	if pb1.Type == "RSA PRIVATE KEY" {
		pb1, pb2 = pb2, pb1
	}
	kpem, err := getByteFromPemBlock(pb2, keypass)
	if err != nil {
		return err
	}
	pb2 = &pem.Block{Type: pb2.Type, Bytes: kpem}
	fz.ClientCert, err = tls.X509KeyPair(pem.EncodeToMemory(pb1), pem.EncodeToMemory(pb2))
	if err != nil {
		return err
	}
	fz.UseClientCert = true
	return nil
}

func getDate(d string, bStart bool) time.Time {
	t, err := time.Parse("2006/01/02", d)
	if err != nil {
		if bStart {
			t = time.Now()
			return t.AddDate(0, -1, 0)
		}
		return time.Now()
	}
	return t
}

// AdminExport : 管理者による設定CSVファイルのエクスポート
func (fz *FzAPI) AdminExport(mode, outfile, sd, ed string) error {
	params := map[string]string{}
	bMB := false
	switch mode {
	case "fzlog":
		params["action"] = "History"
		params["sub_action"] = "do_export"
		sdate := getDate(sd, true)
		edate := getDate(ed, false)
		params["start_year_selected"] = sdate.Format("2006")
		params["start_month_selected"] = sdate.Format("01")
		params["start_day_selected"] = sdate.Format("02")
		params["end_year_selected"] = edate.Format("2006")
		params["end_month_selected"] = edate.Format("01")
		params["end_day_selected"] = edate.Format("02")
		params["folder_selected"] = ""
		params["search_text"] = ""
	case "mblog":
		bMB = true
		sdate := getDate(sd, true)
		edate := getDate(ed, false)
		params["tm_type"] = "mtime"
		params["tm_s_y"] = sdate.Format("2006")
		params["tm_s_c"] = sdate.Format("01")
		params["tm_s_d"] = sdate.Format("02")
		params["tm_s_h"] = "00"
		params["tm_s_m"] = "00"
		params["tm_e_y"] = edate.Format("2006")
		params["tm_e_c"] = edate.Format("01")
		params["tm_e_d"] = edate.Format("02")
		params["tm_e_h"] = "23"
		params["tm_e_m"] = "59"
		params["subj"] = ""
		params["ps"] = ""
		params["ac_new"] = "1"
		params["ac_accepted"] = "1"
		params["ac_declined"] = "1"
		params["ac_canceled"] = "1"
		params["ac_download"] = "1"
		params["ac_upload"] = "1"
		params["ac_delete"] = "1"
		params["limit"] = "20000"
		params["csv"] = "csv"
		params["deleted_user"] = "0"
	case "fzuser":
		params["action"] = "User_export"
		params["sub_action"] = "do_export"
		params["passwd_mode"] = "enc"
	case "fzuser_check":
		params["action"] = "User_export"
		params["sub_action"] = "do_export_uid"
		params["uid_export_mode"] = "check"
	case "fzuser_perm":
		params["action"] = "User_perm_export"
		params["sub_action"] = "do_export"
	case "fzprj":
		params["action"] = "Project_export"
		params["sub_action"] = "do_export"
	case "fzfolder":
		params["action"] = "Group_export"
		params["sub_action"] = "do_export"
	default:
		return fmt.Errorf("Invalid admin export mode")
	}
	if !bMB {
		return fz.FzExportCSV(params, outfile)
	}
	return fz.MbLogExport(params, outfile)
}

// AdminImport : 管理者による設定CSVファイルのインポート
func (fz *FzAPI) AdminImport(mode, infile string) error {
	params := map[string]string{}
	fileKey := "filename"
	switch mode {
	case "fzuser":
		params["action"] = "User_import"
		params["sub_action"] = "do_upload"
		params["passwd_mode"] = "enc"
	case "fzuser_perm":
		params["action"] = "User_perm_import"
		params["sub_action"] = "do_upload"
		fileKey = "filename_user_perm"
	case "fzprj":
		params["action"] = "Project_import"
		params["sub_action"] = "do_upload"
	case "fzfolder":
		params["action"] = "Group_import"
		params["sub_action"] = "do_upload"
	default:
		return fmt.Errorf("Invalid import Type")
	}
	return fz.FzImportCSV(params, infile, fileKey)
}

// MbAdminExport : めるあど便承認者エクスポート
func (fz *FzAPI) MbAdminExport(mode, uid, outfile string) error {
	path := ""
	switch mode {
	case "authority":
		path = "/admin/approve/export/authority"
	case "global":
		path = "/admin/approve/export/global/"
	case "addrbook":
		path = "/job/addrbook/?action=export"
	case "admin_addrbook":
		path = "/admin/addrbook/?action=export&uid=" + uid
	default:
		return fmt.Errorf("Invalid export type")
	}
	return fz.MbExportCSV(path, outfile)
}

// MbAdminImport : めるあど便のCSVファイルインポート
func (fz *FzAPI) MbAdminImport(mode, uid, infile string) error {
	path := ""
	fileKey := "file"
	replace := false
	switch mode {
	case "authority":
		path = "/admin/approve/import/authority"
	case "global":
		path = "/admin/approve/import/global/"
	case "addrbook":
		fileKey = "addrbook_file"
		path = "/job/addrbook/?action=import"
	case "admin_addrbook":
		fileKey = "addrbook_file"
		path = "/admin/addrbook/?action=import"
	case "admin_addrbook_replace":
		fileKey = "addrbook_file"
		path = "/admin/addrbook/?action=import"
		replace = true
	default:
		return fmt.Errorf("Invalid MbAdminImport mode")
	}
	return fz.MbImportCSV(path, uid, infile, fileKey, replace)
}

// FzcConfig : FileZen Client設定ファイルの定義
type FzcConfig struct {
	FzURL        string `json:"FzUrl"`
	FzUID        string `json:"FzUid"`
	FzPassword   string `json:"FzPassword"`
	LocalFolder  string `json:"LocalFolder"`
	FzDownFolder string `json:"FzDownFolder"`
	FzUpFolder   string `json:"FzUpFolder"`
	NotifyTo     string `json:"NotifyTo"`
	NotifyMode   string `json:"NotifyMode"`
}

// LoadFzcConfig : FilZen Clientの設定ファイルを読み込む
func LoadFzcConfig(path, key string) (*FzcConfig, error) {
	config := &FzcConfig{}
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("LoadFzcConfig err=%v", err)
	}
	if err := json.Unmarshal(file, config); err != nil {
		return nil, fmt.Errorf("LoadFzcConfig err=%v", err)
	}
	if config.FzUID != "" && config.FzPassword != "" {
		pass, err := decrypt(config.FzUID+key, config.FzPassword)
		if err == nil {
			config.FzPassword = pass
		}
	}
	return config, nil
}

// SaveFzcConfig : FileZen Client設定ファイルの保存
func SaveFzcConfig(config *FzcConfig, path, key string) error {
	if config.FzUID != "" && config.FzPassword != "" {
		pass, err := encrypt(config.FzUID+key, config.FzPassword)
		if err != nil {
			return fmt.Errorf("SaveFzcConfig err=%v", err)
		}
		config.FzPassword = pass
	}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("SaveFzcConfig err=%v", err)
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("SaveFzcConfig err=%v", err)
	}
	_, err = f.Write(b)
	if err != nil {
		f.Close()
		return fmt.Errorf("SaveFzcConfig err=%v", err)
	}
	f.Close()
	os.Remove(path)
	err = os.Rename(tmp, path)
	if err != nil {
		return fmt.Errorf("SaveFzcConfig err=%v", err)
	}
	return nil
}

// Util Crypto
func getAesKey(keyText string) []byte {
	if len(keyText) > 32 {
		return []byte(keyText[:32])
	}
	for len(keyText) < 32 {
		keyText += "S"
	}
	return []byte(keyText)
}

// encrypt string to base64 crypto using AES
func encrypt(keyText string, text string) (string, error) {
	key := getAesKey(keyText)
	plaintext := []byte(text)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(crand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decrypt from base64 to decrypted string
func decrypt(keyText string, cryptoText string) (string, error) {
	key := getAesKey(keyText)
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)
	return fmt.Sprintf("%s", ciphertext), nil
}
