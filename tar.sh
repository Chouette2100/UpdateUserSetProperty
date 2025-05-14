#! /bin/bash
DATE=`date '+%Y%m%d-%H%M'`
tar zcvf UUSP_$DATE.tar.Z DBConfig.yml UpdateUserSetProperty run.sh my_script.env
