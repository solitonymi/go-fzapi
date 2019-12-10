package fzapi

import (
	"os"
	"testing"
)

// TestFzArchive : アーカイブアクセスライブラリの試験
func TestFzArchive(t *testing.T) {
	folder := os.Getenv("FZ_ARCHIVE_DIR")
	if folder == "" {
		t.Fatal("No Achive Folder")
	}
	a := NewFzArchive()
	if a.LoadFzArchive(folder); len(a.Errors) > 0 {
		for _, e := range a.Errors {
			t.Error(e)
		}
		t.Fatalf("LoadFzArchive has %d errors", len(a.Errors))
	}
	s := &SearchFzArchiveEnt{
		Start:  "2010-01-01",
		End:    "2020-01-02",
		UserID: "admi.*",
	}
	r := a.SearchFzPrjArchiveFile(s)
	if len(r) > 0 {
		t.Errorf("Found invalid data %+v", r[0])
	}
}
