package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"path/filepath"

	"github.com/jhoonb/archivex"
	fzapi "github.com/solitonymi/go-fzapi"
	"github.com/urfave/cli/v2"
)

var logFile *os.File
var config *fzapi.FzcConfig

func setupConf(c *cli.Context) {
	cpath := c.String("config")
	master := c.String("master")
	if cpath != "" {
		conf, err := fzapi.LoadFzcConfig(cpath, master)
		if err != nil {
			log.Fatalf("setupConf err=%v", err)
		}
		config = conf
	} else {
		config = &fzapi.FzcConfig{}
	}
	if c.String("url") != "" {
		config.FzURL = c.String("url")
	}
	if c.String("uid") != "" {
		config.FzUID = c.String("uid")
	}
	if c.String("passwd") != "" {
		config.FzPassword = c.String("passwd")
	}
	if c.String("local") != "" {
		config.LocalFolder = c.String("local")
	}
	if c.String("upload") != "" {
		config.FzUpFolder = c.String("upload")
	}
	if c.String("download") != "" {
		config.FzDownFolder = c.String("download")
	}
	if c.String("notify") != "" {
		config.NotifyMode = c.String("notify")
	}
	if c.String("to") != "" {
		config.NotifyTo = c.String("to")
	}
	logDir := c.String("log")
	if logDir != "" {
		path := filepath.Join(logDir, time.Now().Format("20060102")+".log")
		var err error
		logFile, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatalf("Cannot open log file err=%v", err)
		}
		log.SetOutput(io.MultiWriter(logFile, os.Stderr))
	}
}

func main() {
	os.Exit(Run())
}

// Run : メインルーチン
func Run() int {
	app := cli.NewApp()
	app.Version = "2.0.0"
	app.Name = "fzc"
	app.Usage = "FileZen Client."
	app.Copyright = "(c) 2020 Soliton Systems K.K."
	app.Commands = []*cli.Command{
		{
			Name:  "mkconf",
			Usage: "Make Config File",
			Action: func(c *cli.Context) error {
				setupConf(c)
				return makeFzConfig(c)
			},
		},
		{
			Name:  "test",
			Usage: "Test Login to FileZen",
			Action: func(c *cli.Context) error {
				setupConf(c)
				return testLogin(c)
			},
		},
		{
			Name:  "sync",
			Usage: "Synchronize the folder",
			Action: func(c *cli.Context) error {
				setupConf(c)
				return syncFolder(c)
			},
		},
		{
			Name:  "mbsend",
			Usage: "Send FileZen Mail",
			Action: func(c *cli.Context) error {
				setupConf(c)
				return mbSend(c)
			},
		},
		{
			Name:  "import",
			Usage: "import [fzuser|]",
			Action: func(c *cli.Context) error {
				setupConf(c)
				return adminImport(c)
			},
		},
		{
			Name:  "export",
			Usage: "import [fzuser|] -csv <csv file>",
			Action: func(c *cli.Context) error {
				setupConf(c)
				return adminExport(c)
			},
		},
		{
			Name:  "mbimport",
			Usage: "mbimport [fzuser|] -csv <csv file>",
			Action: func(c *cli.Context) error {
				setupConf(c)
				return mbImport(c)
			},
		},
		{
			Name:  "mbexport",
			Usage: "mbexport [fzuser|] -csv <csv file>",
			Action: func(c *cli.Context) error {
				setupConf(c)
				return mbExport(c)
			},
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "url",
			Usage:    "FileZen `URL`",
			EnvVars:  []string{"FZ_URL"},
			Required: false,
		},
		&cli.StringFlag{
			Name:     "uid",
			Usage:    "FileZen `UID`",
			EnvVars:  []string{"FZ_UID"},
			Required: false,
		},
		&cli.StringFlag{
			Name:     "passwd",
			Usage:    "FileZen `Password`",
			Aliases:  []string{"password"},
			EnvVars:  []string{"FZ_PASSWD"},
			Required: false,
		},
		&cli.StringFlag{
			Name:     "config",
			Usage:    "Config `FILE`",
			Value:    "",
			EnvVars:  []string{"FZC_CONF"},
			Required: false,
		},
		&cli.StringFlag{
			Name:     "master",
			Usage:    "Master `PASSWORD`",
			Value:    "",
			EnvVars:  []string{"FZC_MASTER"},
			Required: false,
		},
		&cli.StringFlag{
			Name:     "cert",
			Usage:    "Client Cert & Key `FILE`",
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "keypass",
			Usage:    "Client Cert Private Key `PASSWORD`",
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:  "notify",
			Usage: "FileZen Notify Mode `DOWNLOAD/ALTER/DELETE`",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "to",
			Usage: "FileZen Notify to `ALL|2|`",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "local",
			Usage: "Local `FOLDER`",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "upload",
			Usage: "FileZen upload `PROJECT/FOLDER`",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "download",
			Usage: "FileZen download `PROJECT/FOLDER`",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "csv",
			Usage: "Import `CSV File`",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mbconf",
			Usage: "FileZen `Mail Config File`",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "out",
			Usage: "FileZen Config outpu `FILE`",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "tuid",
			Usage: "Target `UID`",
			Value: "",
		},
		&cli.StringFlag{
			Name:     "log",
			Usage:    "Log `DIR`",
			Aliases:  []string{"l"},
			Value:    "",
			Required: false,
		},
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

func makeFzConfig(c *cli.Context) error {
	if err := fzapi.SaveFzcConfig(config, c.String("out"), c.String("master")); err != nil {
		return err
	}
	return nil
}

func testLogin(c *cli.Context) error {
	fz, err := loginToFileZen(c)
	if err != nil {
		return err
	}
	defer fz.FzLogout()
	return nil
}

func syncFolder(c *cli.Context) error {
	if config.FzDownFolder == "" && config.FzUpFolder == "" {
		return fmt.Errorf("No FileZen folder to sync")
	}
	if config.LocalFolder == "" {
		return fmt.Errorf("No local Folder")
	}
	// Check Dirs
	if !checkDir("Download") || !checkDir("Upload") || !checkDir("fztmp") {
		return fmt.Errorf("checkDirs Error")
	}
	log.Println("Start FzSync")
	fz, err := loginToFileZen(c)
	if err != nil {
		return err
	}
	defer fz.FzLogout()
	// Download
	d := fz.FzFindFolder(config.FzDownFolder)
	if config.FzDownFolder != "" && d.ID == "" {
		return fmt.Errorf("Download folder not found. %s", config.FzDownFolder)
	}
	for _, f := range d.FileList {
		if f.Key == "" {
			continue
		}
		localfile := filepath.Join(config.LocalFolder, "/Download/", f.Name)
		fstat, err := os.Stat(localfile)
		if err != nil {
			log.Printf("Download Start %s\n", f.Name)
			st := time.Now().Unix()
			err = fz.FzDownload(f.Key, localfile)
			if err == nil {
				et := time.Now().Unix()
				s, err := strconv.ParseInt(f.Size, 10, 64)
				if err != nil {
					s = 0
				}
				dt := et - st
				speed := "-"
				if dt > 0 {
					speed = fmt.Sprintf("%.3fKbps", float64(s)/(1024.0*float64(dt)))
				}
				log.Printf("Download Done %s speed=%s \n", f.Name, speed)
			} else {
				log.Printf("Download Failed %s\n", f.Name)
				fz.FzReload()
			}
		} else {
			if fmt.Sprintf("%d", fstat.Size()) != f.Size {
				log.Printf("File size mismatch %s\n", f.Name)
			}
		}
	}
	// Upload
	uppath := filepath.Join(config.LocalFolder, "/Upload", "/*")
	files, _ := filepath.Glob(uppath)
	ud := fz.FzFindFolder(config.FzUpFolder)
	if config.FzUpFolder != "" && ud.ID == "" {
		return fmt.Errorf("Upload folder not found. %s", config.FzUpFolder)
	}
	if !fz.CanUpload(config.FzUpFolder, "") {
		return fmt.Errorf("Upload Failed No Perimission %s", config.FzUpFolder)
	}
	for _, f := range files {
		f, _ = filepath.Abs(f)
		fstat, err := os.Stat(f)
		bZip := false
		if err != nil {
			log.Printf("Upload Skip stat error %s\n", f)
			continue
		}
		bf := filepath.Base(f)
		if fstat.IsDir() {
			bZip = true
			bf += ".zip"
		}
		if fz.CanUpload(config.FzUpFolder, bf) {
			if bZip {
				f = makeZip(bf, f)
				fstat, err = os.Stat(f)
				if err != nil {
					log.Printf("Skip upload make zip error  %s\n", f)
					continue
				}
			}
			com := getFileComment(f)
			d = fz.FzFindFolder(config.FzUpFolder)
			st := time.Now().Unix()
			if fstat.Size() > 1024*1024*50 {
				err = fz.FzPlUpload(f, d.ID, bf, com, config.NotifyTo, config.NotifyMode)
			} else {
				err = fz.FzUpload(f, d.ID, bf, com, config.NotifyTo, config.NotifyMode)
			}
			if err == nil {
				et := time.Now().Unix()
				dt := et - st
				speed := "-"
				if dt > 0 {
					speed = fmt.Sprintf("%.3fKbps", float64(fstat.Size())/(1024.0*float64(dt)))
				}
				log.Printf("Upload Done %s speed=%s\n", bf, speed)
			} else {
				log.Printf("Upload Failed %s err=%v\n", bf, err)
				fz.FzReload()
			}
			if bZip {
				log.Printf("Delete  temp zip file %s\n", f)
				err = os.Remove(f)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
	log.Println("End Folder Sync")
	return nil
}

func isExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
func checkDir(dir string) bool {
	d := filepath.Join(config.LocalFolder, dir)
	if !isExists(d) {
		log.Printf("Create dir '%s'", d)
		err := os.Mkdir(d, 0777)
		if err != nil {
			log.Println(err)
			return false
		}
	}
	return true
}

func getFileComment(f string) string {
	fstat, err := os.Stat(f)
	ret := ""
	ret += fmt.Sprintf("OrgPath: %s\n", f)
	ret += fmt.Sprintf("Size: %d\n", fstat.Size())
	//
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

func makeZip(bf, f string) string {
	zipfile := filepath.Join(config.LocalFolder, "/fztmp/", bf)
	zipfile, _ = filepath.Abs(zipfile)
	log.Printf("Make Zip file '%s' from dir '%s'", zipfile, f)
	zip := new(archivex.ZipFile)
	zip.Create(zipfile)
	zip.AddAll(f, true)
	zip.Close()
	return zipfile
}

func mbSend(c *cli.Context) error {
	mbconf := c.String("mbconf")
	if mbconf == "" {
		return fmt.Errorf("mbconf is null")
	}
	if _, err := os.Stat(mbconf); err != nil {
		return err
	}
	fz, err := loginToFileZen(c)
	if err != nil {
		return err
	}
	defer fz.FzLogout()
	return fz.FzMail(mbconf)
}

func adminImport(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("Invalid sub command")
	}
	fz, err := loginToFileZen(c)
	if err != nil {
		return err
	}
	defer fz.FzLogout()
	if err := fz.AdminImport(c.Args().Get(0), c.String("csv")); err != nil {
		return err
	}
	return nil
}

func adminExport(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("Invalid sub command")
	}
	fz, err := loginToFileZen(c)
	if err != nil {
		return err
	}
	defer fz.FzLogout()
	if err := fz.AdminExport(c.Args().Get(0), c.String("csv"), c.String("start"), c.String("end")); err != nil {
		return err
	}
	return nil
}

func mbImport(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("Invalid sub command")
	}
	fz, err := loginToFileZen(c)
	if err != nil {
		return err
	}
	defer fz.FzLogout()
	if err := fz.MbAdminImport(c.Args().Get(0), c.String("tuid"), c.String("csv")); err != nil {
		return err
	}
	return nil
}

func mbExport(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("Invalid sub command")
	}
	fz, err := loginToFileZen(c)
	if err != nil {
		return err
	}
	defer fz.FzLogout()
	if err := fz.MbAdminExport(c.Args().Get(0), c.String("tuid"), c.String("csv")); err != nil {
		return err
	}
	return nil
}

func loginToFileZen(c *cli.Context) (*fzapi.FzAPI, error) {
	fz := &fzapi.FzAPI{}
	if c.String("cert") != "" && c.String("keypass") != "" {
		if err := fz.LoadClientCert(c.String("cert"), c.String("keypass")); err != nil {
			return nil, err
		}
	}
	if err := fz.FzLogin(config.FzURL, config.FzUID, config.FzPassword); err != nil {
		return nil, err
	}
	return fz, nil
}
