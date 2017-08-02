#! /bin/bash
#
#cd /utry_workspace/ucAlarm

if [ "$1" == "-h" ]; then
	echo ""
	echo "[USE] ./ucaOption.sh start | startD | stop | restart | restartD | reload | check"
	echo ""
elif [ "$1" == "start" ]; then
	bin/ucAlarm -c conf/ucAlarm.conf &
elif [ "$1" == "startD" ]; then
	bin/ucAlarm -c conf/ucAlarm.conf -debug &
elif [ "$1" == "stop" ]; then
	pkill -9 ucAlarm
elif [ "$1" == "restart" ]; then
	pkill -9 ucAlarm
	sleep 1
	bin/ucAlarm -c conf/ucAlarm.conf &
elif [ "$1" == "restartD" ]; then
	pkill -9 ucAlarm
    sleep 1
    bin/ucAlarm -c conf/ucAlarm.conf -debug &
elif [ "$1" == "reload" ]; then
	bin/ucAlarm -reload
elif [ "$1" == "check" ]; then
	pid=`pgrep ucAlarm`
	if [ ! -n "$pid" ]; then
		bin/ucAlarm -c conf/ucAlarm.conf -debug &
	fi
fi
