package main

import (
	"fmt"
	"log"
	"io"
	"os"
	"time"

	"net/http"

	"github.com/Chouette2100/srapi"
	"github.com/Chouette2100/exsrapi"
	"github.com/Chouette2100/srdblib"
)


const Version = "0.0.1"

// Genreが空のレコードが検索する
func SelectUserWithoutGenre() (
	userlist []int,
	err error,
) {

	//	sqlstmt := "select userno from user where genre = '' limit 3"
	sqlstmt := "select userno from user where genre = '' "
	//	sqlstmt := "select userno from user where genre = 'unknown' "

	stmt, err := srdblib.Db.Prepare(sqlstmt)
	if err != nil {
		err = fmt.Errorf("prepare(): %w", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		err = fmt.Errorf("query(): %w", err)
		return nil, err
	}
	defer rows.Close()

	var userno int
	for rows.Next() {
		err = rows.Scan(&userno)
		if err != nil {
			err = fmt.Errorf("scan: %w", err)
			return nil, err
		}
		userlist = append(userlist, userno)
	}
	return
}

func UpdateUserSetGenre(client *http.Client, userno int) (
	err error,
) {

	//	ユーザーのGenreを取得する
	roominf, err := srapi.ApiRoomProfile(client, fmt.Sprintf("%d", userno))
	if err != nil {
		err = fmt.Errorf("GetUser(%d) returned error. %w", userno, err)
		return err
	}

	if roominf.Genre == "" {
		err = fmt.Errorf("GetUser(%d) returned empty genre", userno)
		return err
	}


	//	ユーザーのGenreを更新する
	sqlstmt := "update user set genre = ? where userno = ?"

	stmt, err := srdblib.Db.Prepare(sqlstmt)	
	if err != nil {
		err = fmt.Errorf("prepare: %w", err)
		return err
	}
	defer stmt.Close()

	genre := srdblib.ConverGenre2Abbr(roominf.Genre)
	_, err = stmt.Exec(genre, userno)
	if err != nil {
		err = fmt.Errorf("exec: %w", err)
		return err
	}
	log.Printf("userno=%d genre=%s longname=%s\n", userno, genre, roominf.Name)
	return
}

// Genreが空のレコードを探す
// 該当するレコードのGenreに値を設定する
func main() {

		//	ログ出力を設定する
		logfile, err := exsrapi.CreateLogfile(Version, srdblib.Version)
		if err != nil {
			panic("cannnot open logfile: " + err.Error())
		}
		defer logfile.Close()
		//	log.SetOutput(logfile)
		log.SetOutput(io.MultiWriter(logfile, os.Stdout))
	
		//	データベースとの接続をオープンする。
		var dbconfig *srdblib.DBConfig
		dbconfig, err = srdblib.OpenDb("DBConfig.yml")
		if err != nil {
			err = fmt.Errorf("srdblib.OpenDb() returned error. %w", err)
			log.Printf("%s\n", err.Error())
			return
		}
		if dbconfig.UseSSH {
			defer srdblib.Dialer.Close()
		}
		defer srdblib.Db.Close()
		log.Printf("********** Dbhost=<%s> Dbname = <%s> Dbuser = <%s> Dbpw = <%s>\n",
			(*dbconfig).DBhost, (*dbconfig).DBname, (*dbconfig).DBuser, (*dbconfig).DBpswd)
	
		//      cookiejarがセットされたHTTPクライアントを作る
		client, jar, err := exsrapi.CreateNewClient("anonymous")
		if err != nil {
			err = fmt.Errorf("CreateNewClient() returned error. %w", err)
			log.Printf("%s\n", err.Error())
			return
		}
		//      すべての処理が終了したらcookiejarを保存する。
		defer jar.Save() //	忘れずに！
	

	// Genreが空のレコードを探す
	userlist, err := SelectUserWithoutGenre()
	if err != nil {
		err = fmt.Errorf("SelectUserWithoutGenre: %w", err)
		log.Printf("%v", err)
		return
	}

	// 該当するレコードのGenreに値を設定する
	for _, user := range userlist {
		err = UpdateUserSetGenre(client, user)
		if err != nil {
			err = fmt.Errorf("UpdateUserSetGenre: %w", err)
			log.Printf("%v", err)
			continue //	エラーの場合は次のレコードへ。
		}
		time.Sleep(time.Second) //	1秒のスリープを入れる。
	}
}
