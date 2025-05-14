package main

import (
	"flag"
	"fmt"
	//	"io"
	"log"
	//	"os"

	//	"strings"
	//	"strconv"
	"time"

	"net/http"

	"github.com/go-gorp/gorp"
	//      "gopkg.in/gorp.v2"

	//	"github.com/dustin/go-humanize"

	"github.com/Chouette2100/exsrapi"
	"github.com/Chouette2100/srapi"
	"github.com/Chouette2100/srdblib"
)

/*

0.0.1 UserでGenreが空白の行のGenreをSHOWROOMのAPIで更新する。
0.0.2 Userでirankが-1の行のランクが空白の行のランク情報をSHOWROOMのAPIで更新する。
0.1.0 DBのアクセスにgorpを導入する。
0.1.1 database/sqlを使った部分（コメント）を削除する
00AA00	実行時パラメータの導入等crontabで使用できるように改修する。
00AB00	実行時パラメータに"user"を追加する。履歴（itrank）の設定方法の誤りを修正する（最大=>最小）
00AC00	パラメータとその設定を実用的なものにする。
00AD00	cmd="entry" を追加する（イベントに参加しているルームをすべてSHOWランク取得の対象とできる）
00AD01	ログ出力先をファイルのみとする。
00AE00	ランキングはsrapi.GetGenreRankingByApi()を使って取得する（2024年6月はじめの仕様変更への対応）
00AE01	コマンドライン変数にiscurrentを追加する。
00AE02	コマンドライン変数にpagesを追加する。
00AE03	InsertまたはUpdateを行ったときだけウェイトするものとし、またウエイトの単位はmsとする。
00AE04	"Entry"のソートをeventuser.point（降順）で行う
00AE05	spmmhhをsphhmmと訂正する。
00AF00	srdblib.Dbmap.AddTableWithName(srdblib.Userhistory{}, "userhistory").SetKeys(false, "Userno", "Ts")を追加する
00AG00	V2.0.0環境で再ビルド

*/

const Version = "00AG00"

//      "gopkg.in/gorp.v2"

// Genreが空のレコードを探す
// 該当するレコードのGenreに値を設定する

// 更新対象とするレコードを検索する。
// 0.0.1 Genreが空のレコードが検索する
// 0.0.2 irankが-1でrankが ' SS|SS-?' または ' S|S-? ' のレコードを検索する。
func SelectFromUserByCond(
	client *http.Client,
	cmd string,
	prd string,
	srlimit int,
	evth int,
	evhhmm int,
	etlimit int,
	ptth int,
	userno int,
	iscurrent bool,
	pages int,
) (
	userlist []interface{},
	err error,
) {

	tnow := time.Now().Truncate(time.Second)
	sqlst := ""

	if cmd != "ranking" {
		//	単純な select文でsql実行後にエラー処理を行うケース
		switch cmd {
		case "user": //	usernoで指定したルームを対象とする。
			/*
				//	sqlst = "select * from user where inrank = 100000 limit 100 "
				sqlst = "select * from user where userno = ? "
				userlist, err = srdblib.Dbmap.Select(srdblib.User{}, sqlst, userno)
				if err != nil {
					err = fmt.Errorf("select(): %w", err)
					return nil, err
				}
			*/
			var user interface{}
			user, err = srdblib.Dbmap.Get(&srdblib.User{}, userno)
			if err != nil {
				err = fmt.Errorf("srdblib.Dbmap.Get(): %w", err)
			} else {
				if user == nil {
					user = &srdblib.User{Userno: userno}
				}
				userlist = append(userlist, user)
			}

		case "showrank": //	showrank
			//	上位ルームに対してデータの再取得を行う
			sqlst = "select * from user where getp is not null and irank > 0 order by irank limit ? "
			userlist, err = srdblib.Dbmap.Select(srdblib.User{}, sqlst, srlimit)
			if err != nil {
				err = fmt.Errorf("select(): %w", err)
				return nil, err
			}
		case "chkoldtype": //	chkoldtype
			sqlst = "select * from user where `rank` in "
			sqlst += "('SS | SS-5', 'SS | SS-4', 'SS | SS-3', 'SS | SS-2', 'SS | SS-1', "
			sqlst += "'S | S-5', 'S | S-4', 'S | S-3', 'S | S-2', 'S | S-1', "
			sqlst += "'A | A-5', 'A | A-4', 'A | A-3', 'A | A-2', 'A | A-1', "
			sqlst += "'B | B-5') "
			userlist, err = srdblib.Dbmap.Select(srdblib.User{}, sqlst)
		case "point": //	point
			//	直近の獲得ポイント上位のルームのランクを再取得する。
			//	tnow := time.Now()
			etime := tnow.Truncate(5 * time.Minute)
			btime := etime.Add(-5 * time.Minute)
			sqlst = "select user_id as userno from points where ts between ? and ? and point > ? "
			sqlst += " order by point desc"
			userlist, err = srdblib.Dbmap.Select(srdblib.User{}, sqlst, btime, etime, ptth)
		case "event": //	event
			tt := tnow
			hh, mm, _ := tt.Clock()
			//	日付を求めるのだが、Truncate()はUTCで切り捨てを行うので09:00JST以前が前日になってしまうことへの対策
			tt = tt.Add(9 * time.Hour).Truncate(24 * time.Hour).Add(-9 * time.Hour)
			if hh*60+mm < evhhmm/100*60+evhhmm%100 {
				//	evhhmmで指定した時刻より前に実行するときは前々日終了のイベントの結果を対象とする。
				tt = tt.Add(-24 * time.Hour)
			}

			tty := tt.Add(-24 * time.Hour)

			sqlstmt := "select eu.userno from eventuser eu join event e on ( eu.eventid = e.eventid ) "
			sqlstmt += " where e.endtime between ? and ? and eu.point > ? order by eu.point desc "
			userlist, err = srdblib.Dbmap.Select(srdblib.User{}, sqlstmt, tty, tt, evth)
		case "entry":
			sqlstmt := "select eu.userno from eventuser eu join event e on eu.eventid = e.eventid "
			sqlstmt += " where curdate() between e.starttime and e.endtime order by eu.point desc limit ? "
			userlist, err = srdblib.Dbmap.Select(srdblib.User{}, sqlstmt, etlimit)
			//	default:
		}
		if err != nil {
			err = fmt.Errorf("select(): %w", err)
			return nil, err
		}

	} else {
		// 	定式的なエラー処理が行えないデータ抽出を行うケース
		switch cmd {
		case "ranking": //	ranking
			//	ランキング上位のランクを取得する。
			genreid := 0

			gr := &[]srapi.GenreRanking{}
			gr, err = srapi.GetGenreRankingByApi(client, genreid, prd, iscurrent, pages)
			if err != nil {
				err = fmt.Errorf("srapi.GetGenreRankingByApi(): %w", err)
				return nil, err
			}
			for _, v := range *gr {
				userlist = append(userlist, &srdblib.User{Userno: v.Room.RoomID})
			}
		}

		//	default:
	}

	//	userlist, err = srdblib.Dbmap.Select(srdblib.User{}, "select weu.userno from weventuser weu join wevent we on ( weu.eventid = we.eventid )  where we.endtime between '2024-05-13' and '2024-05-20' and weu.point > 300000 order by weu.point desc ")
	//	userlist, err = srdblib.Dbmap.Select(srdblib.User{}, "select weu.userno from weventuser weu join wevent we on ( weu.eventid = we.eventid )  where we.endtime between '2024-05-13' and '2024-05-20' and  weu.point = -1 and weu.vld < 5 order by weu.point desc ")

	//	userlist, err = srdblib.Dbmap.Select(srdblib.User{}, )

	//	sqlstmt := "select distinct userno from weventuser where point > ? and userno in (select userno from user where irank = 888888888 ) "
	//	userlist, err = srdblib.Dbmap.Select(srdblib.User{}, sqlstmt, 1500000)

	/*
		userlist = append(userlist, &srdblib.User{Userno: 87991})
	*/

	for i, v := range userlist {
		log.Printf(" user[%d] = %+v\n", i, v.(*srdblib.User).Userno)
	}

	return
}

func main() {

	var (
		cmd    = flag.String("cmd", "showrank", "string flag")
		sphhmm = flag.Int("sphhmm", 0, "int flag")
		wait   = flag.Int("wait", 3000, "int flag")
		prd    = flag.String("prd", "daily", "string flag")
		//	srth    = flag.Int("srth", 350600000, "int flag")
		srlimit   = flag.Int("srlimit", 220, "int flag")
		evth      = flag.Int("evth", 500000, "int flag")
		evhhmm    = flag.Int("evhhmm", 1205, "int flag")
		etlimit   = flag.Int("etlimit", 300, "int flag")
		ptth      = flag.Int("ptth", 100000, "int flag")
		userno    = flag.Int("userno", 0, "int flag")
		iscurrent = flag.Bool("iscurrent", false, "bool flag")
		pages     = flag.Int("pages", 6, "int flag")
	)

	//	ログ出力を設定する
	logfile, err := exsrapi.CreateLogfile(Version, srdblib.Version)
	if err != nil {
		panic("cannnot open logfile: " + err.Error())
	}
	defer logfile.Close()
	log.SetOutput(logfile)
	//	log.SetOutput(io.MultiWriter(logfile, os.Stdout))

	flag.Parse()

	log.Printf("param -cmd : %s\n", *cmd)
	log.Printf("param -sphhmm : %d\n", *sphhmm)
	log.Printf("param -wait : %d\n", *wait)
	log.Printf("param -prd : %s\n", *prd)
	log.Printf("param -srlimit : %d\n", *srlimit)
	log.Printf("param -evth : %d\n", *evth)
	log.Printf("param -evhhmm : %d\n", *evhhmm)
	log.Printf("param -etlimit : %d\n", *etlimit)
	log.Printf("param -ptth : %d\n", *ptth)
	log.Printf("param -userno : %d\n", *userno)
	log.Printf("param -iscurrent : %t\n", *iscurrent)
	log.Printf("param -pages : %d\n", *pages)

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
	srdblib.Dbmap = &gorp.DbMap{Db: srdblib.Db, Dialect: dial, ExpandSliceArgs: true}

	srdblib.Dbmap.AddTableWithName(srdblib.User{}, "user").SetKeys(false, "Userno")
	srdblib.Dbmap.AddTableWithName(srdblib.Userhistory{}, "userhistory").SetKeys(false, "Userno", "Ts")

	//      cookiejarがセットされたHTTPクライアントを作る
	client, jar, err := exsrapi.CreateNewClient("anonymous")
	if err != nil {
		err = fmt.Errorf("CreateNewClient() returned error. %w", err)
		log.Printf("%s\n", err.Error())
		return
	}
	//      すべての処理が終了したらcookiejarを保存する。
	defer jar.Save() //	忘れずに！

	// 	条件に一致するユーザを抽出する。
	userlist, err := SelectFromUserByCond(client, *cmd, *prd, *srlimit, *evth, *evhhmm, *etlimit, *ptth, *userno, *iscurrent, *pages)
	//	userlist, err := SelectFromUserByCond()
	if err != nil {
		err = fmt.Errorf("SelectFromUserByCond(): %w", err)
		log.Printf("%v", err)
		return
	}

	log.Printf("no of userlist = %d\n", len(userlist))

	// 該当するレコードのGenreに値を設定する
	tnow := time.Now().Truncate(time.Second)
	sp := (*sphhmm/100)*60 + (*sphhmm % 100)
	hh, mm, _ := tnow.Clock()
	tn := hh*60 + mm
	if sp > tn {
		sp = sp - 24*60
	}
	pd := tn - sp + 5

	for _, v := range userlist {
		user := v.(*srdblib.User)
		if user.Userno == 0 {
			continue
		}
		err = srdblib.UpinsUserSetProperty(client, tnow, user, pd, *wait)
		if err != nil {
			err = fmt.Errorf("UpinsUserSetProperty(): %w", err)
			log.Printf("%v", err)
			continue //	エラーの場合は次のレコードへ。
		}
		//	time.Sleep(time.Duration(*wait) * time.Millisecond)

	}
}
