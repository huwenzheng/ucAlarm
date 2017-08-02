/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: ucserv.go
@Version	: 5.0.20
===============================================
*/

package ucserv

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	alarm "alarm"
	alarmlog "alarmlog"
	alarmpro "alarmpro"
	alarmsys "alarmsys"
	common "common"
	ucflag "ucflag"
	ucsignal "ucsignal"
)

func sigHandlerFunc(s os.Signal, arg interface{}) {
	switch s {
	case syscall.SIGALRM:
		common.CheckError(errors.New("we got signal SIGALRM"), common.DEBUG)
	case syscall.SIGPIPE:
		common.CheckError(errors.New("we got signal SIGPIPE"), common.DEBUG)
	case syscall.SIGUSR1:
		common.IsReloadOpt = true
		if err := ucflag.InitConfig(); err != nil {
			common.CheckError(errors.New("Reload failed"), common.ERROR)
		} else {
			common.CheckError(errors.New("Reload success"), common.DEBUG)
		}
	}
}

func StartServer() {

	sigHandler := ucsignal.SignalSetNew()
	sigHandler.Register(syscall.SIGALRM, sigHandlerFunc)
	sigHandler.Register(syscall.SIGPIPE, sigHandlerFunc)
	sigHandler.Register(syscall.SIGUSR1, sigHandlerFunc)
	sigChan := make(chan os.Signal, 10)
	signal.Notify(sigChan, syscall.SIGALRM, syscall.SIGPIPE, syscall.SIGUSR1)
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	//start alarm module
	go alarm.AlarmObject_ptr.StartAlarm()

	//start process monit module
	go alarmpro.AlarmPro_Monitor.StartMonit()

	//start monit log module
	go alarmlog.AlarmObject_logMonitor.StartMonit()

	//start monit system module
	go alarmsys.AlarmObject_sysMonitor.StartMonit()

	for {
		select {
		case sig := <-sigChan:
			sigHandler.Handle(sig, nil)
		default:
			time.Sleep(5 * time.Second)

			if !alarm.AlarmObject_ptr.IsRunOk {
				alarm.AlarmObject_ptr.StopAlarm()

				time.Sleep(3 * time.Second)
				alarm.AlarmObject_ptr.IsRunOk = true
				go alarm.AlarmObject_ptr.StartAlarm()
			}

			if !alarmpro.AlarmPro_Monitor.IsRunOk {
				alarmpro.AlarmPro_Monitor.StopMonit()

				time.Sleep(3 * time.Second)
				alarmpro.AlarmPro_Monitor.IsRunOk = true
				go alarmpro.AlarmPro_Monitor.StartMonit()
			}

			if !alarmlog.AlarmObject_logMonitor.IsLogMonitorOk {
				alarmlog.AlarmObject_logMonitor.StopMonit()

				time.Sleep(3 * time.Second)
				alarmlog.AlarmObject_logMonitor.IsLogMonitorOk = true
				go alarmlog.AlarmObject_logMonitor.StartMonit()
			}

			if !alarmsys.AlarmObject_sysMonitor.IsSysMonitOk {
				alarmsys.AlarmObject_sysMonitor.StopMonit()

				time.Sleep(3 * time.Second)
				alarmsys.AlarmObject_sysMonitor.IsSysMonitOk = true
				go alarmsys.AlarmObject_sysMonitor.StartMonit()
			}

			errInfo := fmt.Sprintf("goroutine=%d, cpu=%d", runtime.NumGoroutine(), runtime.NumCPU())
			common.CheckError(errors.New(errInfo), common.DEBUG)
			runtime.GC()
			runtime.Gosched()
		}
	}
}
