package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMkFzcConf : 設定ファイルの作成
func TestFzc(t *testing.T) {
	t.Log("Start")
	tmpFile, err := ioutil.TempFile("", "fzc")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	conf := tmpFile.Name()
	local, err := ioutil.TempDir("", "fzc")
	if err != nil {
		t.Fatal(err)
	}
	// Error Case
	cmd := "fzc -url a -uid b -passwd c -local test -upload test/upload -download test/download test"
	doTest(t, cmd, 1)
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

	// Make FZC Config file
	cmd = fmt.Sprintf("fzc -out %s -url %s -uid %s -passwd %s -local %s -upload test/Upload -download test/Download -master=test mkconf", conf, url, uid, passwd, local)
	doTest(t, cmd, 0)
	defer os.Remove(conf)
	// test command
	cmd = fmt.Sprintf("fzc -config %s -master=test test", conf)
	doTest(t, cmd, 0)

	// mbsend command
	mbconf := makeTestMBConf(email, t)
	defer os.Remove(mbconf)
	cmd = fmt.Sprintf("fzc -config %s -master=test -mbconf %s mbsend", conf, mbconf)
	doTest(t, cmd, 0)
	// sync
	cmd = fmt.Sprintf("fzc -config %s -master=test sync", conf)
	os.MkdirAll(filepath.Join(local, "Download"), 0770)
	os.MkdirAll(filepath.Join(local, "Upload"), 0770)
	doTest(t, cmd, 0)
	t.Log("Done")
}

func doTest(t *testing.T, cmd string, code int) {
	os.Args = strings.Split(cmd, " ")
	if r := Run(); r != code {
		t.Errorf("cmd=%s code %d", cmd, r)
	}
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
	tmpFile.WriteString("file:../../testdata/test.txt\n")
	tmpFile.WriteString(fmt.Sprintf("start:%s\n", time.Now().Format("2006/01/02")))
	tmpFile.WriteString("days:1\n")
	tmpFile.WriteString("limit:1\n")
	tmpFile.WriteString("comment:Test\n")
	return tmpFile.Name()
}
