package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	prompt "github.com/c-bata/go-prompt"
	fzapi "github.com/solitonymi/go-fzapi"
)

var fza *fzapi.FzArchive
var suggestions = []prompt.Suggest{
	// Command
	{Text: "search", Description: "seach file list Ex. fza>serach mode start end uid filename"},
	{Text: "save", Description: "save file Ex. fza> save mode filepath outdir"},
	{Text: "exit", Description: "Exit"},
	// Mode
	{Text: "prj", Description: "Project Mode"},
	{Text: "mb", Description: "FileZen Mail Mode"},
	{Text: "rf", Description: "Recv Folder Mode"},
}

func executor(in string) {
	in = strings.TrimSpace(in)

	blocks := strings.Split(in, " ")
	switch blocks[0] {
	case "exit":
		fmt.Println("Good Bye!")
		os.Exit(0)
	case "search":
		serach(blocks)
		return
	case "save":
		save(blocks)
		return
	}
	fmt.Println("Sorry, I don't understand.")
}

func completer(in prompt.Document) []prompt.Suggest {
	w := in.GetWordBeforeCursor()
	if w == "" {
		return []prompt.Suggest{}
	}
	return prompt.FilterHasPrefix(suggestions, w, true)
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage:fza <archive folder>")
	}
	fza = fzapi.NewFzArchive()
	if fza.LoadFzArchive(os.Args[1]); len(fza.Errors) > 0 {
		for _, e := range fza.Errors {
			log.Println(e)
		}
		log.Fatalf("LoadFzArchive Failed")
	}
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("fza> "),
	)
	p.Run()
}

func serach(blocks []string) {
	if len(blocks) < 6 {
		log.Printf("Invalid Params")
		return
	}
	mode := blocks[1]
	if mode != "prj" && mode != "mb" && mode != "rf" {
		log.Printf("Invalid Mode %s", mode)
		return
	}
	s := &fzapi.SearchFzArchiveEnt{
		Start:    blocks[2],
		End:      blocks[3],
		UserID:   blocks[4],
		FileName: blocks[5],
	}
	if mode == "prj" {
		r := fza.SearchFzPrjArchiveFile(s)
		for _, e := range r {
			ts := time.Unix(0, e.TimeStamp)
			fmt.Printf("%v,%s,%s,%s,%s,%s\n", ts, e.UserID, e.ProjectName, e.FolderName, e.FileName, e.FilePath)
		}
		return
	}
	r := fza.SearchFzMbRfArchiveFile(s)
	for _, e := range r {
		ts := time.Unix(0, e.TimeStamp)
		fmt.Printf("%v,%s,%s,%s,%s\n", ts, e.UserID, e.JobID, e.FileName, e.FilePath)
	}
}

func save(blocks []string) {
	if len(blocks) < 4 {
		log.Printf("Invalid Params")
		return
	}
	mode := blocks[1]
	if mode != "prj" && mode != "mb" && mode != "rf" {
		log.Printf("Invalid Mode %s", mode)
		return
	}
	if mode == "prj" {
		if err := fza.SaveFzPrjArchiveFile(blocks[2], blocks[3]); err != nil {
			log.Println(err)
		}
		return
	}
	if err := fza.SaveFzMbRfArchiveFile(blocks[2], blocks[3]); err != nil {
		log.Println(err)
	}
}
