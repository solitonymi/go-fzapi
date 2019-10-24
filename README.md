# go-fzapi

FileZen API for golang

[![Godoc Reference](https://godoc.org/github.com/solitonymi/go-fzapi?status.svg)](http://godoc.org/github.com/solitonymi/go-fzapi)
[![Go Report Card](https://goreportcard.com/badge/solitonymi/go-fzapi)](https://goreportcard.com/report/solitonymi/go-fzapi)

FileZenをGO言語から利用するためのAPIライブラリです。
Soliton Systems K.K.の公式リリースではないため、
サポートは、githubのみで行っております。

## 使用方法

### インストール

```go
	import snkweb "github.com/solitonymi/go-fzapi"
```

### ログイン

```go
	fz := &fzapi.FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		log.Fatalf("FzLogin err=%v", err)
	}
```

### フォルダの取得

’プロジェクト名/フォルダ名’で指定します。

```go
	f := fz.FzFindFolder("パブリック/パブリック")
	if f == nil {
		log.Error("FzFindFolder パブリック/パブリック フォルダがありません。")
	}
```

### アップロード

アップロード先は、フォルダの取得で取得したIDを使用します。

```go
	if err := fz.FzPlUpload(filepath.Join("testdata", "test.txt"), f.ID, "test.txt", "test", "","");err != nil {
		log.Errorf("FzUpload err=%v", err)
	}
```

### ファイルのダウンロード

ファイルのキーを取得して、ダウンロードを行います。

```go
	if key := fz.FzFindFile("パブリック", "パブリック", "test.txt"); key != "" {
		if err := fz.FzDownload(key, filepath.Join("testdata", "test2.txt")); err != nil {
			log.Errorf("FzDownload err=%v", err)
		}
	} else {
		log.Error("FzFindFolder パブリック/パブリックに test.txtがありません。 ")
	}
```

### ファイルを削除する

ファイルのキーを取得して、削除を行います。

```go
	if key := fz.FzFindFile("パブリック", "パブリック", "test.txt"); key != "" {
		if err := fz.FzDeleteFile(key); err != nil {
			log.Errorf("FzDeleteFile err=%v", err)
		}
	} else {
		log.Error("FzFindFolder パブリック/パブリックに test.txtがありません。 ")
	}
```

### ログアウト

```go
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
```

### めるあど便の送信

以下のフォーマットのめるあど便送信ファイルを作成します。

```
subject:件名
mailto:宛先メールアドレス
from:送信元メールアドレス
file:送信するファイルまたは、フォルダ（フォルダの場合は、ZIP圧縮します。）
start:公開開始日（ex. 2019/10/23)
days:公開日数
limit:ダウンロード回数
comment:コメント
コメントの続き...
|
```

このファイルを指定して、APIをコールします。

```go
	fz := &fzapi.FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		log.Fatalf("FzLogin err=%v", err)
	}
	if err := fz.FzMail("めるあど便送信ファイル名"); err != nil {
		log.Fatalf("FzMail err=%v", err)
	}
	if err := fz.FzLogout(); err != nil {
		log.Fatalf("FzLogout err=%v", err)
	}
```

###  履歴などのCSVファイルのダウンロード

```go
	fz := &fzapi.FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		log.Fatalf("FzLogin err=%v", err)
	}
	et := time.Now() // 終了日
	st := et.Add(-time.Hour * 24) // 開始日
	if err := fz.AdminExport("fzlog", "ダウンロードファイル名", st.Format("2006/01/02"), et.Format("2006/01/02")); err != nil {
		log.Errorf("AdminExport mode %s err=%v", m, err)
	}
	if err := fz.FzLogout(); err != nil {
		log.Fatalf("FzLogout err=%v", err)
	}
```

AdminExportの第一パラメータは、ダウンロードするファイルの種別です。
履歴に関しては、期間の指定が必要です。
ファイルの種別には、以下のものがあります。

|種別|内容|
|---|---|
|fzlog|プロジェクトの履歴|
|mblog|めるあど便の履歴|
|fzuser|登録ユーザーリスト|
|fzuser_check|ユーザーのチェックリスト |
|fzuser_perm|ユーザー権限のリスト|
|fzprj|プロジェクトのリスト|
|fzfolder|フォルダのリスト|

### CSVによる登録

```go
	fz := &fzapi.FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		log.Fatalf("FzLogin err=%v", err)
	}
	if err := fz.AdminImport("fzprj", "CSVファイル名"); err != nil {
		log.Errorf("AdminImport err=%v", err)
	}
	if err := fz.FzLogout(); err != nil {
		t.Fatalf("FzLogout err=%v", err)
	}
```

AdminImportの第一パラメータは、インポートするCSVの種別です。以下の種別の対応しています。

|種別|内容|
|---|---|
|fzuser|登録ユーザーリスト|
|fzuser_perm|ユーザー権限のリスト|
|fzprj|プロジェクトのリスト|
|fzfolder|フォルダのリスト|

インポートするファイルの仕様は、FileZenのWeb画面からアップロードするファイルと同じです。
公式マニュアルを参照ください。

### めるあど便の承認者、アドレス帳のエクスポート

```go
	fz := &fzapi.FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		log.Fatalf("FzLogin err=%v", err)
	}
	if err := fz.MbAdminExport("authority", "", "ダウンロードCSVファイル名"); err != nil {
		log.Errorf("MbAdminExport mode %s err=%v", m, err)
	}
	if err := fz.FzLogout(); err != nil {
		log.Fatalf("FzLogout err=%v", err)
	}
```

MbAdminExportの第一パラメータは、エクスポートするCSVの種別です。種別には、以下のものがあります。

|種別|内容|
|---|---|
|authority|ユーザー毎に割り当てる承認者|
|global|全体に割り当てる承認者リスト|
|addrbook|ログインしたユーザーのアドレス帳|
|admin_addrbook|管理者権限で、ユーザーのアドレス帳をダウンロードする。対象のユーザーは、第二引数で指定します。|

### めるあど便の承認者、アドレス帳のインポート

```go
	fz := &fzapi.FzAPI{}
	if err := fz.FzLogin(url, uid, passwd); err != nil {
		log.Fatalf("FzLogin err=%v", err)
	}
	if err := fz.MbAdminImport("addrbook", "", csv); err != nil {
		log.Errorf("AdminImport err=%v", err)
	}
	if err := fz.FzLogout(); err != nil {
		log.Fatalf("FzLogout err=%v", err)
	}
```

MbAdminImportの第一引数は、インポートするCSVの種別です。
種別は、エクスポートと同じです。

### クライアント証明書の変換

PEMまたは、PKCS#12の証明書の

```go
	if err := ImportClientCert("インポートする証明書のファイル名", "インポートパスワード", "出力する証明書のファイル名", "出力する証明書のパスワード"); err != nil {
		log.Fatal(err)
	}
```

### クライアント証明書の利用

ログインする前に、LoadClientCertにより証明書を読み込みます。

```go
	fz := &fzapi.FzAPI{}
	if err := fz.LoadClientCert("変換した証明書", "パスワード"); err != nil {
		log.Fatal(err)
	}
```

### FileZen Client設定ファイルの保存

```go
	config := &fzapi.FzcConfig{
		FzURL:        "http://10.30.100.168",
		FzPassword:   "admin",
		FzUID:        "admin",
		FzDownFolder: "test/download",
		FzUpFolder:   "test/upload",
		NotifyMode:   "ALL",
		NotifyTo:     "ALL",
		LocalFolder:  "test",
	}
	if err := fzapi.SaveFzcConfig(config, "fzc.json", "Password"); err != nil {
		log.Fatal(err)
	}
```

### FileZen Client設定ファイルの読み込み

```go
	config, err := LoadFzcConfig("fzc.json", "Password")
	if err != nil {
		log.Fatal(err)
	}
```

## ユニットテスト

ユニットテストのためには、FileZenに関する情報が必要です。
これは、以下の環境変数で、指定します。

|環境変数|内容|
|---|---|
|FZ_URL|FileZenのURL|
|FZ_UID|FileZenにアクセスするユーザーID|
|FZ_PASSWD|ユーザーIDに対応したパスワード|

## インストール

```
$ go get github.com/solitonymi/go-fzapi
```

# ライセンス

Apache 2.0

#  作者

Masayuki Yamai

Soliton Systems K.K 
