#! /bin/bash

# 起動方法
#
# run.sh コマンド [パラメータ] コマンド [パラメータ] コマンド [パラメータ] ...


# "コマンド [パラメータ]" の例

# 既存のSHOWランクの順にしたがってデータを取得する
# Sr データ数（220） 

# データ取得対象となっているイベントの参加ルームのデータを取得する
# Et データ数（300） 

# 前日（または前々日）に終了したイベントの上位ルームからデータを取得する
# Ev ポイント下限（500000）

# 現在の獲得ポイント取得対象ルームの獲得ポイントの大きいルームからデータを取得する
# Pt データ数 （100000）

# 指定したルームIDのルームのデータを取得する
# Us ルームID

# 指定したルームIDのルームのデータを取得する
# Rk 期間（daily, weekly, monthly, annually, all_time） 現在/前期間（current/last） ページ数（ページ数*20がデータ数）(5〜20 期間によって調整)

cd /home/chouette/MyProject/Showroom/UpdateUserSetProperty

export DBNAME=xxxxxxxxxx
export DBUSER=xxxxxxxxxx
export DBPW=xxxxxxxxxx


# 最初のコマンドで更新されたデータは2番目以降のコマンドでは更新されないようにするため処理の最初にHHMMを設定する
# 現在のデータのタイムスタンプがこの時刻以後のものは更新と対象としない
# 処理をまたがって後続のコマンドでは更新されないようにするには外部でHHMMを定義しておく。
if [ -z $HHMM ]; then
    DATE=`date '+%H%M'`
    HHMM=$DATE
fi

# 時刻が"0"で始まっていると8進数とみなされるので、その場合10進数への変換が必要
# 以下のいずれの方法でもよい
#HHMM=`expr $HHMM + 0`
HHMM=$((10#$HHMM))

#echo $HHMM

# 取得ごとのウェイト時間（ms） （アクセス制限がある模様、2000でもいいかも）
WT=3000

LM=UpdateUserSetProperty

#for arg; do
#  case ${arg} in

i=1
while [ "$#" -gt 0 ]; do
	dt=`date`
	echo $dt ${1} >> UUSP.log
	echo $dt ${1} >> UUSP.err
  case $1 in
  Sr )
    ./$LM -cmd showrank -srlimit $2 -spmmhh $HHMM -wait $WT >> UUSP.log 2>> UUSP.err
    shift
    ;;
  Et )
    ./$LM -cmd entry -etlimit $2 -spmmhh $HHMM -wait $WT >> UUSP.log 2>> UUSP.err
    shift
    ;;
  Ev )
    ./$LM -cmd event -evth $2 -evhhmm 1205 -spmmhh $HHMM -wait $WT >> UUSP.log 2>> UUSP.err
    shift
    ;;
  Pt )
    ./$LM -cmd point -ptth $2 -spmmhh $HHMM -wait $WT >> UUSP.log 2>> UUSP.err
    shift
    ;;
  Us )
    ./$LM -cmd user -userno $2 -spmmhh $HHMM -wait $WT >> UUSP.log 2>> UUSP.err
    shift
    ;;
  Rk )
    if [ $3 = "current"  ]; then
    ./$LM -cmd ranking -prd $2 -pages $4 -iscurrent true -spmmhh $HHMM -wait $WT >> UUSP.log 2>> UUSP.err
    elif [ $3 = "last" ]; then
    ./$LM -cmd ranking -prd $2 -pages $4 -spmmhh $HHMM -wait $WT >> UUSP.log 2>> UUSP.err
    else
	    echo unknown parameter $3 - must be current or last >> UUSP.log
    fi
    shift
    shift
    shift
    ;;
  * )
    echo unknown command $1 >> UUSP.log
    break
  esac
  shift
done
