package fzapi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFzAPIProject : プロジェクトの試験
func TestFzAPIProject(t *testing.T) {
	url := os.Getenv("FZ_URL")
	if url == "" {
		t.Fatal("No Url")
	}
	uid := os.Getenv("FZ_UID")
	if uid == "" {
		t.Fatal("No UID")
	}
	passwd := os.Getenv("FZ_PASSWD")
	if passwd == "" {
		t.Fatal("No Password")
	}
	fz := &FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		t.Fatalf("FzLogin err=%v", err)
	}
	if err := fz.FzReload(); err != nil {
		t.Errorf("FzReload err=%v", err)
	}
	// パブリック/パプリック フォルダを探す
	f := fz.FzFindFolder("パブリック/パブリック")
	if f == nil {
		t.Error("FzFindFolder パブリック/パブリック フォルダがありません。")
	} else {
		if fz.CanUpload("パブリック/パブリック", "test.txt") {
			if err := fz.FzUpload(filepath.Join("testdata", "test.txt"), f.ID, "test.txt", FzGetFileComment("testdata/test.txt"), "", ""); err != nil {
				t.Errorf("FzUpload err=%v", err)
			} else {
				if err := fz.FzReload(); err != nil {
					t.Errorf("FzReload err=%v", err)
				}
				if fz.CanUpload("パブリック/パブリック", "test.txt") {
					t.Error("CanUpload アップロード可能チェック試験失敗")
				}
				if key := fz.FzFindFile("パブリック", "パブリック", "test.txt"); key != "" {
					if err := fz.FzDownload(key, filepath.Join("testdata", "test2.txt")); err != nil {
						t.Errorf("FzDownload err=%v", err)
					}
					defer os.Remove(filepath.Join("testdata", "test2.txt"))
					if err := fz.FzDeleteFile(key); err != nil {
						t.Errorf("FzDeleteFile err=%v", err)
					}
				} else {
					t.Error("FzFindFolder パブリック/パブリックに test.txtがありません。 ")
				}
			}
		} else {
			t.Error("FzFindFolder パブリック/パブリックにtest.txtをアップロードできません。")
		}
	}
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
	t.Log("Done")
}

// TestFzAPIProjectPLUpload : 分割アップロードの試験
func TestFzAPIProjectPLUpload(t *testing.T) {
	url := os.Getenv("FZ_URL")
	if url == "" {
		t.Fatal("No Url")
	}
	uid := os.Getenv("FZ_UID")
	if uid == "" {
		t.Fatal("No UID")
	}
	passwd := os.Getenv("FZ_PASSWD")
	if passwd == "" {
		t.Fatal("No Password")
	}
	fz := &FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		t.Fatalf("FzLogin err=%v", err)
	}
	// パブリック/パプリック フォルダを探す
	f := fz.FzFindFolder("パブリック/パブリック")
	if f == nil {
		t.Error("FzFindFolder パブリック/パブリック フォルダがありません。")
	} else {
		if fz.CanUpload("パブリック/パブリック", "test.txt") {
			if err := fz.FzPlUpload(filepath.Join("testdata", "test.txt"), f.ID, "test.txt", "test", "", ""); err != nil {
				t.Errorf("FzUpload err=%v", err)
			} else {
				if err := fz.FzReload(); err != nil {
					t.Errorf("FzReload err=%v", err)
				}
				if key := fz.FzFindFile("パブリック", "パブリック", "test.txt"); key != "" {
					if err := fz.FzDownload(key, filepath.Join("testdata", "test2.txt")); err != nil {
						t.Errorf("FzDownload err=%v", err)
					}
					defer os.Remove(filepath.Join("testdata", "test2.txt"))
					if err := fz.FzDeleteFile(key); err != nil {
						t.Errorf("FzDeleteFile err=%v", err)
					}
				} else {
					t.Error("FzFindFolder パブリック/パブリックに test.txtがありません。 ")
				}
			}
		} else {
			t.Error("FzFindFolder パブリック/パブリックにtest.txtをアップロードできません。")
		}
	}
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
	t.Log("Done")
}

// TestFzAPIFileZenMail : めるあど便の試験
func TestFzAPIFileZenMail(t *testing.T) {
	url := os.Getenv("FZ_URL")
	if url == "" {
		t.Fatal("No Url")
	}
	uid := os.Getenv("FZ_UID")
	if uid == "" {
		t.Fatal("No UID")
	}
	passwd := os.Getenv("FZ_PASSWD")
	if passwd == "" {
		t.Fatal("No Password")
	}
	email := os.Getenv("FZ_EMAIL")
	if email == "" {
		t.Fatal("No EMAIL")
	}
	mbconf := makeTestMBConf(email, t)
	fz := &FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		t.Fatalf("FzLogin err=%v", err)
	}
	if err := fz.FzMail(mbconf); err != nil {
		t.Errorf("FzMail err=%v", err)
	}
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
	t.Log("Done")
}

// makeTestMBConf : めるあど便送信のための設定ファイルを作成する
func makeTestMBConf(email string, t *testing.T) string {
	tmpFile, err := ioutil.TempFile("", "mbconftest")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	tmpFile.WriteString("subject:Test\n")
	tmpFile.WriteString(fmt.Sprintf("mailto:%s\n", email))
	tmpFile.WriteString(fmt.Sprintf("from:%s\n", email))
	tmpFile.WriteString("file:./testdata/test.txt\n")
	tmpFile.WriteString(fmt.Sprintf("start:%s\n", time.Now().Format("2006/01/02")))
	tmpFile.WriteString("days:1\n")
	tmpFile.WriteString("limit:1\n")
	tmpFile.WriteString("comment:Test\n")
	return tmpFile.Name()
}

func TestFzAPIAdminExport(t *testing.T) {
	url := os.Getenv("FZ_URL")
	if url == "" {
		t.Fatal("No Url")
	}
	uid := os.Getenv("FZ_UID")
	if uid == "" {
		t.Fatal("No UID")
	}
	passwd := os.Getenv("FZ_PASSWD")
	if passwd == "" {
		t.Fatal("No Password")
	}
	tmpFile, err := ioutil.TempFile("", "adminexport")
	if err != nil {
		t.Fatal(err)
	}
	csv := tmpFile.Name()
	tmpFile.Close()
	os.Remove(csv)
	fz := &FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		t.Fatalf("FzLogin err=%v", err)
	}
	et := time.Now()
	st := et.Add(-time.Hour * 24)
	for _, m := range []string{
		"fzlog", "mblog", "fzuser", "fzuser_check", "fzuser_perm", "fzprj", "fzfolder",
	} {
		if err := fz.AdminExport(m, csv, st.Format("2006/01/02"), et.Format("2006/01/02")); err != nil {
			t.Errorf("AdminExport mode %s err=%v", m, err)
		}
		os.Remove(csv)
	}
	// Bad Mode Test
	if err := fz.AdminExport("bad", csv, st.Format("2006/01/02"), et.Format("2006/01/02")); err == nil {
		t.Errorf("AdminExport mode bad is no err")
	}
	// Test For getDate()
	if err := fz.AdminExport("fzlog", csv, "", ""); err != nil {
		t.Errorf("AdminExport getDate test err=%v", err)
	}
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
	t.Log("Done")
}

func TestFzAPIAdminImport(t *testing.T) {
	url := os.Getenv("FZ_URL")
	if url == "" {
		t.Fatal("No Url")
	}
	uid := os.Getenv("FZ_UID")
	if uid == "" {
		t.Fatal("No UID")
	}
	passwd := os.Getenv("FZ_PASSWD")
	if passwd == "" {
		t.Fatal("No Password")
	}
	csv := makePrjCSV(t)
	defer os.Remove(csv)
	fz := &FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		t.Fatalf("FzLogin err=%v", err)
	}
	if err := fz.AdminImport("fzprj", csv); err != nil {
		t.Errorf("AdminImport err=%v", err)
	}
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
	t.Log("Done")
}

func makePrjCSV(t *testing.T) string {
	tmpFile, err := ioutil.TempFile("", "fzprjcsv")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	tmpFile.WriteString(fmt.Sprintf(",API Test%d,0\n", time.Now().UnixNano()))
	return tmpFile.Name()
}

func TestFzAPIMbAdminExport(t *testing.T) {
	url := os.Getenv("FZ_URL")
	if url == "" {
		t.Fatal("No Url")
	}
	uid := os.Getenv("FZ_UID")
	if uid == "" {
		t.Fatal("No UID")
	}
	passwd := os.Getenv("FZ_PASSWD")
	if passwd == "" {
		t.Fatal("No Password")
	}
	tmpFile, err := ioutil.TempFile("", "adminexport")
	if err != nil {
		t.Fatal(err)
	}
	csv := tmpFile.Name()
	tmpFile.Close()
	os.Remove(csv)
	fz := &FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		t.Fatalf("FzLogin err=%v", err)
	}
	for _, m := range []string{
		"authority", "global", "addrbook", "admin_addrbook",
	} {
		if err := fz.MbAdminExport(m, "admin", csv); err != nil {
			t.Errorf("MbAdminExport mode %s err=%v", m, err)
		}
		os.Remove(csv)
	}
	// Bad Mode Test
	if err := fz.MbAdminExport("bad", "admin", csv); err == nil {
		t.Errorf("AdminExport mode bad is no err")
	}
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
	t.Log("Done")
}

func TestFzAPIMbAdminImport(t *testing.T) {
	url := os.Getenv("FZ_URL")
	if url == "" {
		t.Fatal("No Url")
	}
	uid := os.Getenv("FZ_UID")
	if uid == "" {
		t.Fatal("No UID")
	}
	passwd := os.Getenv("FZ_PASSWD")
	if passwd == "" {
		t.Fatal("No Password")
	}
	csv := makeAddrbook(t)
	defer os.Remove(csv)
	fz := &FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		t.Fatalf("FzLogin err=%v", err)
	}
	if err := fz.MbAdminImport("addrbook", "", csv); err != nil {
		t.Errorf("AdminImport err=%v", err)
	}
	if err := fz.MbAdminImport("admin_addrbook", "test", csv); err != nil {
		t.Errorf("AdminImport err=%v", err)
	}
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
	t.Log("Done")
}

func makeAddrbook(t *testing.T) string {
	tmpFile, err := ioutil.TempFile("", "mbaddrbook")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()
	tmpFile.WriteString("name,test@test.example,group\n")
	return tmpFile.Name()
}

func TestFzAPIClientCert(t *testing.T) {
	if err := ImportClientCert(filepath.Join("testdata", "testin.pem"), "test1234", filepath.Join("testdata", "testout.pem"), "test"); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filepath.Join("testdata", "testout.pem"))
	fz := &FzAPI{}
	if err := fz.LoadClientCert(filepath.Join("testdata", "testout.pem"), "test"); err != nil {
		t.Fatal(err)
	}
	// Bad Cert File Name
	if err := ImportClientCert("tesin.pem", "test", filepath.Join("testdata", "testout.pem"), "test"); err == nil {
		t.Fatal(err)
	}
	// Bad Key Password
	if err := ImportClientCert(filepath.Join("testdata", "testin.pem"), "test", filepath.Join("testdata", "testout.pem"), "test"); err == nil {
		t.Fatal(err)
	}
}

func TestFzcConfig(t *testing.T) {
	c := &FzcConfig{
		FzURL:        "http://10.30.100.168",
		FzPassword:   "admin",
		FzUID:        "admin",
		FzDownFolder: "test/download",
		FzUpFolder:   "test/upload",
		NotifyMode:   "ALL",
		NotifyTo:     "AUTO",
		LocalFolder:  "test",
	}
	path := filepath.Join("testdata", "fzc.json")
	if err := SaveFzcConfig(c, path, "FileZenRA"); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	cs, err := LoadFzcConfig(path, "FileZenRA")
	if err != nil {
		t.Fatal(err)
	}
	if cs.FzPassword != "admin" {
		t.Errorf("Password save error saved password=%s", cs.FzPassword)
	}
}
