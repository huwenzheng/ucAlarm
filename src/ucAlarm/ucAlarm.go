/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: ucAlarm.go
@Version	: 5.0.20
===============================================
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	common "common"
	ucflag "ucflag"
	ucserv "ucserv"
)

func main() {
	flag.Parse()

	if common.Pversion {
		fmt.Printf(" Version %s\n", Version)
		os.Exit(0)
	}

	if common.IsReloadOpt {
		common.DoCommand("pkill -USR1 ucAlarm")
		os.Exit(0)
	}

	if common.IsDebug {
		common.DebugFile, _ = os.Create("logs/ucAlarm.debug")
	}

	if err := common.GetNetworkIp(); err != nil {
		common.CheckError(err, common.ERROR)
		os.Exit(-1)
	}

	if err := ucflag.InitConfig(); err != nil {
		errInfo := fmt.Sprintf("%s(Path: %s)", err.Error(), common.UcAlarmConfPath)
		common.CheckError(errors.New(errInfo), common.ERROR)
		os.Exit(-1)
	}

	common.CheckError(errors.New("Start ucAlarm"), common.INFO)
	defer common.CheckError(errors.New("Stop ucAlarm"), common.INFO)

	ucserv.StartServer()
}
