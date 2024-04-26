package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"net/http"

	"github.com/go-gorp/gorp"
	//      "gopkg.in/gorp.v2"

	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi"
	"github.com/Chouette2100/srapi"
	"github.com/Chouette2100/srdblib"
)

/*

0.0.1 UserでGenreが空白の行のGenreをSHOWROOMのAPIで更新する。
0.0.2 Userでirankが-1の行のランクが空白の行のランク情報をSHOWROOMのAPIで更新する。
0.1.0 DBのアクセスにgorpを導入する。

*/

const Version = "0.1.0"

//      "gopkg.in/gorp.v2"

type User struct {
	Userno       int
	Userid       string
	User_name    string
	Longname     string
	Shortname    string
	Genre        string
	Rank         string
	Nrank        string
	Prank        string
	Irank        int
	Inrank       int
	Iprank       int
	Itrank       int
	Level        int
	Followers    int
	Fans         int
	Fans_lst     int
	Ts           time.Time
	Getp         string
	Graph        string
	Color        string
	Currentevent string
}

// 更新対象とするレコードを検索する。
// 0.0.1 Genreが空のレコードが検索する
// 0.0.2 irankが-1でrankが ' SS|SS-?' または ' S|S-? ' のレコードを検索する。
func SelectUserWithoutGenre() (
	userlist []User,
	err error,
) {

	//	rows, err := Dbmap.Select(User{}, "select * from user where userno = ? ", "164614")
	//	rows, err := Dbmap.Select(User{}, "select * from user where `rank` = ? ", "A-5")
	rows, err := Dbmap.Select(User{}, "select * from user where userno = ? ", "87911")
	if err != nil {
		err = fmt.Errorf("select(): %w", err)
		return nil, err
	}

	for i, v := range rows {
			log.Printf(" user[%d] = %+v\n", i, v.(*User).Userno)
			userlist = append(userlist, *v.(*User))
	}


	return
}

// Rank情報からランクのソートキーを作る
func MakeSortKeyOfRank(rank string, nextscore int) (
	irank int,
) {
	r2n := map[string]int{
		"SS-5":    50000000,
		"SS-4":    60000000,
		"SS-3":    70000000,
		"SS-2":    80000000,
		"SS-1":    90000000,
		"S-5":     150000000,
		"S-4":     160000000,
		"S-3":     170000000,
		"S-2":     180000000,
		"S-1":     190000000,
		"A-5":     250000000,
		"A-4":     260000000,
		"A-3":     270000000,
		"A-2":     280000000,
		"A-1":     290000000,
		"B-5":     350000000,
		"B-4":     360000000,
		"B-3":     370000000,
		"B-2":     380000000,
		"B-1":     390000000,
		"C-10":    400000000,
		"C-9":     410000000,
		"C-8":     420000000,
		"C-7":     430000000,
		"C-6":     440000000,
		"C-5":     450000000,
		"C-4":     460000000,
		"C-3":     470000000,
		"C-2":     480000000,
		"C-1":     490000000,
		"unknown": 1000000000, //	SHOWROOMのアカウントを削除した配信者さん
	}

	if sk, ok := r2n[rank]; ok {
		irank = sk + nextscore
	} else {
		irank = 999999999 //	(アイドルで)SHOWRANKの対象ではない配信者さん
	}

	return
}

func UpdateUserSetRank(client *http.Client, tnow time.Time, user *User) (
	err error,
) {

	//	ユーザーのランク情報を取得する
	ria, err := srapi.ApiRoomProfile_All(client, fmt.Sprintf("%d", user.Userno))
	if err != nil {
		err = fmt.Errorf("ApiRoomProfile_All(%d) returned error. %w", user.Userno, err)
		return err
	}
	if ria.Errors != nil {
		//	err = fmt.Errorf("ApiRoomProfile_All(%d) returned error. %v", userno, ria.Errors)
		//	return err
		ria.ShowRankSubdivided = "unknown"
		ria.NextScore = 0
		ria.PrevScore = 0
	}

	if ria.ShowRankSubdivided == "" {
		err = fmt.Errorf("ApiRoomProfile_All(%d) returned empty.ShowRankSubdivided", user.Userno)
		return err
	}

                user.User_name = ria.RoomName
                user.Genre = ria.GenreName
                user.Rank = ria.ShowRankSubdivided
                user.Nrank = humanize.Comma(int64(ria.NextScore))
                user.Prank = humanize.Comma(int64(ria.PrevScore))
                user.Irank = MakeSortKeyOfRank(ria.ShowRankSubdivided, ria.NextScore)
                user.Inrank = ria.NextScore
                user.Iprank = ria.PrevScore
                user.Itrank = 0
                user.Level = ria.RoomLevel
                user.Followers = ria.FollowerNum
                //	user.Fans = 
                //	user.Fans_lst = 
                user.Ts = tnow
				eurl := ria.Event.URL
				eurla := strings.Split(eurl, "/")
                user.Currentevent = eurla[len(eurla)-1]

        cnt, err := Dbmap.Update(user)
        if err != nil {
                log.Printf("error! %v", err)
                return
        }
        log.Printf("cnt = %d\n", cnt)

	
	log.Printf("userno=%d rank=%s nscore=%d pscore=%d longname=%s\n", user.Userno, ria.ShowRankSubdivided, ria.NextScore, ria.PrevScore, ria.RoomName)
	return
}

// Genreが空のレコードを探す
// 該当するレコードのGenreに値を設定する
var Dbmap *gorp.DbMap
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

	//	gorpの初期設定を行う
	dial := gorp.MySQLDialect{Engine: "InnoDB", Encoding: "utf8mb4"}
	Dbmap = &gorp.DbMap{Db: srdblib.Db, Dialect: dial, ExpandSliceArgs: true}

	Dbmap.AddTableWithName(User{}, "user").SetKeys(false, "Userno")

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
	tnow := time.Now().Truncate(time.Second)
	for _, user := range userlist {
		if user.Userno == 0 {
			continue
		}
		err = UpdateUserSetRank(client, tnow, &user)
		if err != nil {
			err = fmt.Errorf("UpdateUserSetGenre: %w", err)
			log.Printf("%v", err)
			continue //	エラーの場合は次のレコードへ。
		}
		time.Sleep(2 * time.Second) //	1秒のスリープを入れる。
	}
}
