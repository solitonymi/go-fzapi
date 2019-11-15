package main

// コマンドラインから、FileZen Clientの設定ファイルを作成する

import (
	"flag"
	"log"
	"os"

	fzapi "github.com/solitonymi/go-fzapi"
)

// MasterKey : パスワード保存のためのキー
// Config ファイルを利用するプログラムと合わせる必要がある。
const MasterKey = "FileZenRA"

var config = fzapi.FzcConfig{}
var path string

func init() {
	flag.StringVar(&path, "config", "", "FileZen client config file path")
	flag.StringVar(&path, "c", "", "FileZen client config file path")
	flag.StringVar(&config.FzURL, "url", "", "FileZen URL")
	flag.StringVar(&config.FzUID, "uid", "", "FileZen user ID")
	flag.StringVar(&config.FzPassword, "passwd", "", "FileZen password")
	flag.StringVar(&config.LocalFolder, "local", "", "FileZen local folder")
	flag.StringVar(&config.FzUpFolder, "upload", "", "FileZen upload project/folder")
	flag.StringVar(&config.FzDownFolder, "download", "", "FileZen download project/folder")
	flag.StringVar(&config.NotifyMode, "notifymode", "", "FileZen Notify Mode DOWNLOAD|DELETE|")
	flag.StringVar(&config.NotifyTo, "notifyto", "", "FileZen Notify Mail to ALL|AUTO")
}

func main() {
	os.Exit(Run())
}

// Run : 実際の処理
// os.Exit()を１箇所で行い、deferでクリーナップを行うために関数にしている。
// みんなのGO言語のノウハウ
func Run() int {
	flag.Parse()
	if path == "" {
		log.Println("-config or -c arg not found")
		flag.Usage()
		return 1
	}
	if config.FzURL == "" {
		log.Println("-url arg not found")
		flag.Usage()
		return 1
	}
	if config.FzUID == "" {
		log.Println("-uid arg not found")
		flag.Usage()
		return 1
	}
	if config.FzPassword == "" {
		log.Println("-passwd arg not found")
		flag.Usage()
		return 1
	}
	if config.LocalFolder == "" {
		log.Println("-local arg not found")
		flag.Usage()
		return 1
	}
	if err := fzapi.SaveFzcConfig(&config, path, MasterKey); err != nil {
		log.Println(err)
		return 2
	}
	return 0
}
