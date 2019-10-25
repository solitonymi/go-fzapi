package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/solitonymi/go-fzapi"
)

// TestMkFzcConf : 設定ファイルの作成
func TestMkFzcConf(t *testing.T) {
	t.Log("Start")
	tmpFile, err := ioutil.TempFile("", "fzcconftest")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	path := tmpFile.Name()
	// Error Case
	var errList = []struct {
		cmd  string
		code int
	}{
		{cmd: "mkfzcconf -url a -uid b -passwd c -local test -upload test/upload -download test/download", code: 1},
		{cmd: "mkfzcconf -c " + path + " -uid b -passwd c -local test -upload test/upload -download test/download", code: 1},
		{cmd: "mkfzcconf -c " + path + " -url a -passwd c -local test -upload test/upload -download test/download", code: 1},
		{cmd: "mkfzcconf -c " + path + " -url a -uid b -local test -upload test/upload -download test/download", code: 1},
		{cmd: "mkfzcconf -c " + path + " -url a -uid b -passwd c -upload test/upload -download test/download", code: 1},
		{cmd: "mkfzcconf -c /bad/path/bad/bad.txt -url a -uid b -passwd c -local test -upload test/upload -download test/download", code: 2},
	}
	for _, e := range errList {
		config = fzapi.FzcConfig{}
		os.Args = strings.Split(e.cmd, " ")
		if r := Run(); r != e.code {
			t.Errorf("cmd=%s code %d!=%d", e.cmd, e.code, r)
		}
	}
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
	config = fzapi.FzcConfig{}
	os.Args = strings.Split(
		fmt.Sprintf("mkfzcconf -c %s -url %s -uid %s -passwd %s -local test -upload test/upload -download test/download",
			path, url, uid, passwd), " ")
	if r := Run(); r != 0 {
		t.Errorf("Run return code=%d", r)
	}
	defer os.Remove(path)
	c, err := fzapi.LoadFzcConfig(path, MasterKey)
	if err != nil {
		t.Fatal(err)
	}
	if c.FzPassword != passwd {
		t.Errorf("Password Missmatch %s!=%s", passwd, c.FzPassword)
	}
	t.Log("Done")
}
