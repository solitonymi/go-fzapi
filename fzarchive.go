package fzapi

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FzArchive : FileZenアーカイブを管理する
type FzArchive struct {
	FzPrjArcList  []*FzArcPrjEnt
	FzMbRfArcList []*FzArcMbRfEnt
	Errors        []string
}

// FzArcPrjEnt : プロジェクトのアーカイブエントリー
type FzArcPrjEnt struct {
	XMLName      xml.Name `xml:"archive_file"`
	ArcType      string   `xml:"type,attr"`
	FolderID     string   `xml:"folder_id,attr"`
	FileID       string   `xml:"file_id,attr"`
	UserID       string   `xml:"user_id,attr"`
	ProjectName  string   `xml:"project_name,attr"`
	FolderName   string   `xml:"folder_name,attr"`
	FileName     string   `xml:"file_name,attr"`
	TimeStampStr string   `xml:"time_stamp,attr"`
	TimeStamp    int64
	Size         string `xml:"size,attr"`
	FilePath     string
}

// FzArcMbRfEnt : めるあど便、受け取りフォルダのアーカイブエントリー
type FzArcMbRfEnt struct {
	XMLName      xml.Name `xml:"archive_file"`
	ArcType      string   `xml:"type,attr"`
	JobID        string   `xml:"job_id,attr"`
	FileID       string   `xml:"file_id,attr"`
	UserID       string   `xml:"user_id,attr"`
	FileOwner    string   `xml:"file_owner,attr"`
	FileName     string   `xml:"file_name,attr"`
	TimeStampStr string   `xml:"time_stamp,attr"`
	TimeStamp    int64
	Size         string `xml:"size,attr"`
	FilePath     string
}

// SearchFzArchiveEnt : 検索パラメータの構造体
type SearchFzArchiveEnt struct {
	Start             string // 検索の開始日時 (2006-01-02)
	nStart            int64
	End               string // 検索の終了日時(2006-01-02)
	nEnd              int64
	UserID            string // 検索するユーザーIDの正規表現
	userIDRegexp      *regexp.Regexp
	FileName          string // 検索するファイル名の正規表現
	fileNameRegexp    *regexp.Regexp
	ProjectName       string // 検索するプロジェクト名の正規表現
	projectNameRegexp *regexp.Regexp
	FolderName        string // 検索するフォルダ名の正規表現
	folderNameRegexp  *regexp.Regexp
	JobID             string // 検索するJOB ID先頭数文字でOK
}

// NewFzArchive : FileZenのアーカイブを検索する構造体を作成する
func NewFzArchive() *FzArchive {
	return &FzArchive{}
}

// LoadFzArchive : アーカイブフォルダの検索
func (a *FzArchive) LoadFzArchive(folder string) {
	files, _ := filepath.Glob(folder + "/*_archive")
	for _, f := range files {
		a.loadOneModeFzArcFolder(f)
	}
}

func setupSearchParam(s *SearchFzArchiveEnt) {
	var err error
	var t time.Time
	if t, err = time.Parse("2006-01-02", s.Start); err == nil {
		s.nStart = t.UnixNano()
	} else {
		s.nStart = 0
	}
	if t, err = time.Parse("2006-01-02", s.End); err == nil {
		s.nEnd = t.Add(time.Hour * 24).UnixNano()
	} else {
		s.nEnd = time.Now().UnixNano()
	}
	if s.UserID != "" {
		if s.userIDRegexp, err = regexp.Compile(s.UserID); err != nil {
			s.userIDRegexp = nil
		}
	}
	if s.FileName != "" {
		if s.fileNameRegexp, err = regexp.Compile(s.FileName); err != nil {
			s.fileNameRegexp = nil
		}
	}
	if s.ProjectName != "" {
		if s.projectNameRegexp, err = regexp.Compile(s.ProjectName); err != nil {
			s.projectNameRegexp = nil
		}
	}
	if s.FolderName != "" {
		if s.folderNameRegexp, err = regexp.Compile(s.FolderName); err != nil {
			s.folderNameRegexp = nil
		}
	}
}

// SearchFzPrjArchiveFile : プロジェクトのアーカイブからファイルを検索する
func (a *FzArchive) SearchFzPrjArchiveFile(s *SearchFzArchiveEnt) []*FzArcPrjEnt {
	ret := []*FzArcPrjEnt{}
	setupSearchParam(s)
	for _, e := range a.FzPrjArcList {
		if e.TimeStamp < s.nStart {
			continue
		}
		if e.TimeStamp > s.nEnd {
			continue
		}
		if s.fileNameRegexp != nil && !s.fileNameRegexp.MatchString(e.FileName) {
			continue
		}
		if s.projectNameRegexp != nil && !s.projectNameRegexp.MatchString(e.ProjectName) {
			continue
		}
		if s.folderNameRegexp != nil && !s.folderNameRegexp.MatchString(e.FolderName) {
			continue
		}
		ret = append(ret, e)
	}
	return ret
}

// SaveFzPrjArchiveFile : プロジェクトのアーカイブからファイルを復元する
func (a *FzArchive) SaveFzPrjArchiveFile(srcPath, dstPath string) error {
	for _, e := range a.FzPrjArcList {
		if e.FilePath == srcPath {
			src, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer src.Close()
			dst, err := os.Create(filepath.Join(dstPath, e.FileName))
			if err != nil {
				return err
			}
			defer dst.Close()
			_, err = io.Copy(dst, src)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("File not found %s", srcPath)
}

// SearchFzMbRfArchiveFile : めるあど便/受け取りフォルダのアーカイブからファイルを検索する
func (a *FzArchive) SearchFzMbRfArchiveFile(s *SearchFzArchiveEnt) []*FzArcMbRfEnt {
	ret := []*FzArcMbRfEnt{}
	setupSearchParam(s)
	for _, e := range a.FzMbRfArcList {
		if e.TimeStamp < s.nStart {
			continue
		}
		if e.TimeStamp > s.nEnd {
			continue
		}
		if s.fileNameRegexp != nil && !s.fileNameRegexp.MatchString(e.FileName) {
			continue
		}
		if s.JobID != "" && !strings.HasPrefix(e.JobID, s.JobID) {
			continue
		}
		ret = append(ret, e)
	}
	return ret
}

// SaveFzMbRfArchiveFile : めるあど便、受け取りフォルダのアーカイブからファイルを復元する
func (a *FzArchive) SaveFzMbRfArchiveFile(srcPath, dstPath string) error {
	for _, e := range a.FzMbRfArcList {
		if e.FilePath == srcPath {
			src, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer src.Close()
			dst, err := os.Create(filepath.Join(dstPath, e.FileName))
			if err != nil {
				return err
			}
			defer dst.Close()
			_, err = io.Copy(dst, src)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("File not found %s", srcPath)
}

func getTimeStamp(s string) int64 {
	i, err := strconv.Atoi(s)
	if err != nil {
		i = 0
	}
	return time.Unix(int64(i), 0).UnixNano()
}

// loadFzPrjArcEnt :
func (a *FzArchive) loadFzPrjArcEnt(f string) {
	d, err := ioutil.ReadFile(f)
	if err != nil {
		a.Errors = append(a.Errors, fmt.Sprintf("loadFzPrjArcEnt file:%s err:%v", f, err))
		return
	}
	ent := &FzArcPrjEnt{}
	err = xml.Unmarshal(d, ent)
	if err != nil {
		a.Errors = append(a.Errors, fmt.Sprintf("loadFzPrjArcEnt file:%s err:%v", f, err))
		return
	}
	ent.TimeStamp = getTimeStamp(ent.TimeStampStr)
	pos := strings.LastIndex(f, ".")
	if pos > 0 {
		f = f[:pos]
	}
	ent.FilePath = f
	a.FzPrjArcList = append(a.FzPrjArcList, ent)
	return
}

func (a *FzArchive) loadFzMbRfArcEnt(f string) {
	d, err := ioutil.ReadFile(f)
	if err != nil {
		a.Errors = append(a.Errors, fmt.Sprintf("loadFzMbRfArcEnt file:%s err:%v", f, err))
		return
	}
	ent := &FzArcMbRfEnt{}
	err = xml.Unmarshal(d, ent)
	if err != nil {
		a.Errors = append(a.Errors, fmt.Sprintf("loadFzMbRfArcEnt file:%s err:%v", f, err))
		return
	}
	ent.TimeStamp = getTimeStamp(ent.TimeStampStr)
	pos := strings.LastIndex(f, ".")
	if pos > 0 {
		f = f[:pos]
	}
	ent.FilePath = f
	a.FzMbRfArcList = append(a.FzMbRfArcList, ent)
	return
}

func (a *FzArchive) loadOneDayFzArcFolder(folder string) {
	files, _ := filepath.Glob(folder + "/*.xml")
	for _, f := range files {
		if strings.Index(f, "fz_a") != -1 {
			a.loadFzPrjArcEnt(f)
		} else if strings.Index(f, "mb_a") != -1 {
			a.loadFzMbRfArcEnt(f)
		} else if strings.Index(f, "rf_a") != -1 {
			a.loadFzMbRfArcEnt(f)
		}
	}
}

func (a *FzArchive) loadOneModeFzArcFolder(folder string) {
	files, _ := filepath.Glob(folder + "/*-*-*")
	for _, f := range files {
		a.loadOneDayFzArcFolder(f)
	}
}
