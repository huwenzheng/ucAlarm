/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: ucflag.go
@Version	: 5.0.20
===============================================
*/

package ucflag

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	alarm "alarm"
	alarmlog "alarmlog"
	alarmpro "alarmpro"
	alarmsys "alarmsys"
	common "common"
)

var moduleName string //临时保存模块名

func init() {
	moduleName = ""

	flag.BoolVar(&common.Pversion, "version", false, "Print version and exit.")
	flag.BoolVar(&common.IsReloadOpt, "reload", false, "Reload ucAlarm.conf.")
	flag.BoolVar(&common.IsDebug, "debug", false, "Open debug module.")
	flag.StringVar(&common.UcAlarmConfPath, "c", "", "Specify the ucAlarm.conf's path.")

	common.DoCommand("mkdir -p logs")
}

func cleanOldMap() {
	alarm.AlarmObject_ptr.AlarmConf.Mutex.Lock()
	alarmpro.AlarmPro_Monitor.ProConfMap.Mutex.Lock()
	alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.Mutex.Lock()

	defer alarm.AlarmObject_ptr.AlarmConf.Mutex.Unlock()
	defer alarmpro.AlarmPro_Monitor.ProConfMap.Mutex.Unlock()
	defer alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.Mutex.Unlock()

	for k, _ := range alarm.AlarmObject_ptr.AlarmConf.ConfMap {
		delete(alarm.AlarmObject_ptr.AlarmConf.ConfMap, k)
	}
	for k, _ := range alarmpro.AlarmPro_Monitor.ProConfMap.ConfMap {
		delete(alarmpro.AlarmPro_Monitor.ProConfMap.ConfMap, k)
	}
	for k, _ := range alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.ConfMap {
		delete(alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.ConfMap, k)
	}
	for k, _ := range alarmsys.AlarmObject_sysMonitor.SysMonitConfMap.ConfMap {
		delete(alarmsys.AlarmObject_sysMonitor.SysMonitConfMap.ConfMap, k)
	}

	common.IsReloadOpt = false
}

func readLine(r *bufio.Reader) (nRet int, nErr error, first, second string) {
	b, _, err := r.ReadLine()
	if err != nil {
		if err == io.EOF {
			nRet = 0
			nErr = nil
			return
		}
		nRet = 2
		nErr = err
		return
	}

	s := strings.TrimSpace(string(b))
	if strings.Index(s, "#") == 0 {
		nRet = 1
		nErr = nil
		return
	}

	n1 := strings.Index(s, "[")
	n2 := strings.Index(s, "]")
	if n1 > -1 && n2 > -1 && n2 > n1+1 {
		moduleName = s[n1+1 : n2]
		nRet = 1
		nErr = nil
		return
	}

	if len(moduleName) == 0 {
		nRet = 1
		nErr = nil
		return
	}

	index := strings.Index(s, "=")
	if index < 0 {
		nRet = 1
		nErr = nil
		return
	}
	first = strings.TrimSpace(s[:index])
	first = strings.Trim(first, "\t")
	if len(first) == 0 {
		nRet = 1
		nErr = nil
		return
	}
	second = strings.TrimSpace(s[index+1:])
	second = strings.Trim(second, "\t")

	pos := strings.Index(second, "\t#")
	if pos > -1 {
		second = second[0:pos]
	}
	pos = strings.Index(second, " #")
	if pos > -1 {
		second = second[0:pos]
	}
	pos = strings.Index(second, "#")
	if pos > -1 {
		second = second[0:pos]
	}

	second = strings.Replace(second, "\t", " ", -1)
	second = strings.TrimRight(second, " ")
	if len(second) == 0 {
		nRet = 1
		nErr = nil
		return
	}
	nRet = 3
	nErr = nil
	return
}

func configSystemMonit(first, second string) {

	value, ok := alarmsys.AlarmObject_sysMonitor.SysMonitConfMap.ConfMap[first]
	if ok && strings.ToLower(first) == "sendto" {
		second = second[1 : len(second)-1]
		second += ";" + value
	} else {
		second = second[1 : len(second)-1]
	}

	if len(second) > 0 {
		alarmsys.AlarmObject_sysMonitor.SysMonitConfMap.ConfMap[first] = second
	}

}

func configLogParse(first, second string) {

	value, ok := alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.ConfMap[first]
	if ok && strings.ToLower(first) == "sendto" {
		second = second[1 : len(second)-1]
		second += ";" + value
	} else {
		second = second[1 : len(second)-1]
	}
	if len(second) > 0 {
		alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.ConfMap[first] = second
	}

}

func configProcessAlarm(first, second string) {

	value, ok := alarmpro.AlarmPro_Monitor.ProConfMap.ConfMap[first]
	if ok && strings.ToLower(first) == "sendto" {
		second = second[1 : len(second)-1]
		second += ";" + value
	} else {
		second = second[1 : len(second)-1]
	}
	if len(second) > 0 {
		alarmpro.AlarmPro_Monitor.ProConfMap.ConfMap[first] = second
	}

}

func configAlarmEmail(first, second string) {

	second = second[1 : len(second)-1]
	if len(second) > 0 {
		alarm.AlarmObject_ptr.AlarmConf.ConfMap[first] = second
	}

}

func configDefault(first, second string) {

	//监控重启的部分
	value, ok := alarmpro.AlarmPro_Monitor.ProConfMap.ConfMap[first]
	if ok && strings.ToLower(first) == "sendto" {
		second = second[1 : len(second)-1]
		second += ";" + value
	} else {
		second = second[1 : len(second)-1]
	}
	if len(second) > 0 {
		alarmpro.AlarmPro_Monitor.ProConfMap.ConfMap[first] = second
	}

	//日志监控部分
	value, ok = alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.ConfMap[first]
	if ok && strings.ToLower(first) == "sendto" {
		second += ";" + value
	}
	if len(second) > 0 {
		alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.ConfMap[first] = second
	}

	//系统监控部分
	value, ok = alarmsys.AlarmObject_sysMonitor.SysMonitConfMap.ConfMap[first]
	if ok && strings.ToLower(first) == "sendto" {
		second += ";" + value
	}
	if len(second) > 0 {
		alarmsys.AlarmObject_sysMonitor.SysMonitConfMap.ConfMap[first] = second
	}

}

func readInclude(fileName, types string) {

	if fileName[:1] == "\"" {
		fileName = fileName[1 : len(fileName)-1]
	}

	if fileName[:4] == "conf" {
		fileName = common.WorkPath + "/" + fileName
	} else if fileName[:6] == "./conf" {
		fileName = common.WorkPath + fileName[1:]
	}

	fInclude, errInclude := os.Open(fileName)
	if errInclude != nil {
		common.CheckError(errInclude, common.ERROR)
		return
	}
	defer fInclude.Close()

	rInclude := bufio.NewReader(fInclude)
	for {
		nRet_I, err_I, first_I, second_I := readLine(rInclude)
		if nRet_I == 0 {
			return
		} else if nRet_I == 1 {
			continue
		} else if nRet_I == 2 {
			common.CheckError(err_I, common.ERROR)
		} else {
			if types == "default" {
				configDefault(first_I, second_I)
			} else if types == "alarmemail" {
				configAlarmEmail(first_I, second_I)
			} else if types == "processalarm" {
				configProcessAlarm(first_I, second_I)
			} else if types == "logparse" {
				configLogParse(first_I, second_I)
			} else if types == "systemmonit" {
				configSystemMonit(first_I, second_I)
			}
		}
	}
}

func InitConfig() error {

	f, err := os.Open(common.UcAlarmConfPath)
	if err != nil {
		return errors.New("Please specify the path of the ucAlarm.conf or the path is err.")
	}
	defer f.Close()

	if common.IsReloadOpt == true {
		cleanOldMap()
	}

	r := bufio.NewReader(f)
	for {
		nRet, err, first, second := readLine(r)
		if nRet == 0 {
			break
		} else if nRet == 1 {
			continue
		} else if nRet == 2 {
			return err
		}

		switch strings.ToUpper(moduleName) {
		case "DEFAULT":

			if strings.ToLower(first) == "include" {
				readInclude(second, "default")
			} else {
				configDefault(first, second)
			}

		case "ALARM EMAIL":

			if strings.ToLower(first) == "include" {
				readInclude(second, "alarmemail")
			} else {
				configAlarmEmail(first, second)
			}

		case "PROCESS ALARM":

			if strings.ToLower(first) == "include" {
				readInclude(second, "processalarm")
			} else {
				configProcessAlarm(first, second)
			}

		case "LOG PARSE":

			if strings.ToLower(first) == "include" {
				readInclude(second, "logparse")
			} else {
				configLogParse(first, second)
			}

		case "SYSTEM MONIT":

			if strings.ToLower(first) == "include" {
				readInclude(second, "systemmonit")
			} else {
				configSystemMonit(first, second)
			}

		default:
			return fmt.Errorf("not support %s module", moduleName)
		}
	}

	errInfo := fmt.Sprintf("IP: %v\n", common.NetworkIP)
	common.CheckError(errors.New(errInfo), common.DEBUG)

	errInfo = fmt.Sprintf("[alarm center]:	%v\n", alarm.AlarmObject_ptr.AlarmConf.ConfMap)
	common.CheckError(errors.New(errInfo), common.DEBUG)

	errInfo = fmt.Sprintf("[alarm process]: %v\n", alarmpro.AlarmPro_Monitor.ProConfMap.ConfMap)
	common.CheckError(errors.New(errInfo), common.DEBUG)

	errInfo = fmt.Sprintf("[alarm log]: %v\n", alarmlog.AlarmObject_logMonitor.LogMonitorConfMap.ConfMap)
	common.CheckError(errors.New(errInfo), common.DEBUG)

	errInfo = fmt.Sprintf("[alarm sysmonit]: %v\n", alarmsys.AlarmObject_sysMonitor.SysMonitConfMap.ConfMap)
	common.CheckError(errors.New(errInfo), common.DEBUG)

	//将配置下发到各个模块
	if err := alarm.AlarmObject_ptr.InitConf(); err != nil {
		return err
	}
	if err := alarmpro.AlarmPro_Monitor.InitConf(); err != nil {
		return err
	}
	if err := alarmlog.AlarmObject_logMonitor.InitConf(); err != nil {
		return err
	}
	if err := alarmsys.AlarmObject_sysMonitor.InitConf(); err != nil {
		return err
	}

	return nil
}
