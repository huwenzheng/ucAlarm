/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: alarmpro.go
@Version	: 5.0.20
===============================================
*/

package alarmpro

import (
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	alarm "alarm"
	common "common"
	fsnotify "fsnotify"
	uctime "uctime"
)

var AlarmPro_Monitor *monitor //单例模式

type monitor struct {
	watcher       *fsnotify.Watcher //文件监控器
	ProConfMap    common.Config     //保存进程监控的配置文件数据
	sevNameMap    common.Config     //保存服务名和获取对应进程id的命令
	monitLogPath  string            //monit 日志文件路径
	sendTo        string            //进程监控信息接收者
	monitKeyWord  string            //监控信息阀值
	lastAlarmData string            //最后一条报警数据

	IsNeedToChk bool //监控缓冲
	IsconfigOk  bool //配置是否加载正常
	IsRunOk     bool //日志重启监控模块是否运行正常
	IsNeedStop  bool //是否需要停止

	_mutex_1 sync.Mutex   //对lastAlarmData的同步锁
	_mutex_3 sync.RWMutex //读写锁提供对共享变量的线程安全
	_mutex_4 sync.RWMutex //监控缓冲读写锁
}

const (
	KEYWORD = "trying to restart" //默认告警信息关键字
)

func (self *monitor) proParse(lastAlarmData string) {
	self._mutex_3.RLock()
	defer self._mutex_3.RUnlock()

	var proName string
	var err error

	reg := regexp.MustCompile(`\'(.*)\'`)
	strFindArray := reg.FindAllString(lastAlarmData, -1)
	if len(strFindArray) == 0 {
		errInfo := fmt.Sprintf("%s (not found process name)", lastAlarmData)
		common.CheckError(errors.New(errInfo), common.ERROR)
		return
	}
	if proName, err = getFormatName(strFindArray[0]); err != nil {
		errInfo := fmt.Sprintf("%s(%s)", lastAlarmData, err.Error())
		common.CheckError(errors.New(errInfo), common.ERROR)
		return
	}

	if proName == "u3c_config" || proName == "system_ipcc" || proName == "reclaim_log" {
		return
	}

	reg2 := regexp.MustCompile(`(\d+)\:(\d+)\:(\d+)`)
	str_find2 := reg2.FindAllString(lastAlarmData, -1)

	var alarmData string
	alarmInfo, err := common.DoCommand(fmt.Sprintf("cat %s | grep '%s'", self.monitLogPath, str_find2[0]))
	if err != nil {
		common.CheckError(err, common.ERROR)
		return
	} else {
		alarmData = common.ProImagFilter(&alarmInfo, proName)
	}

	var node common.AlarmData
	node.To += self.sendTo
	node.Level += common.WARN
	node.Body += "[IP: " + common.NetworkIP + "]\n\n"
	node.Body += "重启信息:\n"
	node.Body += alarmData
	node.Mailtype += common.TEXT
	node.Subject = common.PRORE + " " + proName + " Restart"

	doCommands := self.proGetCommand(proName)
	if doCommands == "Not Found!" {
		errInfo := fmt.Sprintf("%s command not found! use 'pgrep %s'", proName, proName)
		common.CheckError(errors.New(errInfo), common.WARN)
		doCommands = "pgrep " + proName
	}

	var newPid string
	for i := 0; i < 10; i++ {
		time.Sleep(time.Duration(3) * time.Second)
		result, _ := common.DoCommand(doCommands)
		if result != "" {
			newPid = result
			break
		} else {
			continue
		}
	}
	if newPid != "" {
		restartInfo := fmt.Sprintf("[%s]: Restart Success!\n", uctime.GetFormatDate())
		node.Body += "\n"
		node.Body += restartInfo
		node.Body += "New Pid List:\n"
		node.Body += newPid
	} else {
		restartInfo := fmt.Sprintf("[%s]: Restart more than 30 seconds, Maybe failed!\n", proName)
		node.Body += "\n"
		node.Body += restartInfo
	}

	if alarm.AlarmObject_ptr.IsRunOk == true {
		alarm.AlarmObject_ptr.MsgCh <- node
	}
}

func (self *monitor) proModify() {
	self._mutex_3.RLock()
	defer self._mutex_3.RUnlock()

	alarmInfo, err := common.DoCommand(fmt.Sprintf("cat %s | grep '%s' | grep %s",
		self.monitLogPath, self.monitKeyWord, uctime.GetMonth_str()))
	if err != nil {
		common.CheckError(err, common.ERROR)
		return
	}

	alarmInfoList := strings.Split(alarmInfo, "\n")
	if len(alarmInfoList) < 2 {
		return
	} else {

		self._mutex_1.Lock()
		if self.lastAlarmData == "" {
			self.lastAlarmData = alarmInfoList[len(alarmInfoList)-2]
			self._mutex_1.Unlock()
		} else {
			if self.lastAlarmData != alarmInfoList[len(alarmInfoList)-2] {
				self.lastAlarmData = alarmInfoList[len(alarmInfoList)-2]
				self._mutex_1.Unlock()
				self.proParse(alarmInfoList[len(alarmInfoList)-2])
			} else {
				self._mutex_1.Unlock()
			}
		}
		return
	}
}

func (self *monitor) doMonit() {
	timeout := time.NewTimer(200 * time.Millisecond)
	for {
		select {
		case w := <-self.watcher.Event:
			if w.IsModify() {

				self._mutex_4.Lock()
				if self.IsNeedToChk == true {
					self.IsNeedToChk = false
					go self.proModify()
				}
				self._mutex_4.Unlock()

				continue
			}
			if w.IsRename() {
				self.IsRunOk = false
				return
			}
			if w.IsDelete() {
				errInfo := fmt.Sprintf("%s is Deleted!", self.monitLogPath)
				common.CheckError(errors.New(errInfo), common.WARN)
				self.IsRunOk = false
				return
			}
		case err := <-self.watcher.Error:
			common.CheckError(err, common.ERROR)
			self.IsRunOk = false
			return
		case <-timeout.C:
			timeout.Reset(200 * time.Millisecond)
			if self.IsNeedStop == true {
				self.watcher.Close()
				self.IsRunOk = false
				return
			}

			self._mutex_4.Lock()
			self.IsNeedToChk = true
			self._mutex_4.Unlock()
		}
		runtime.GC()
		runtime.Gosched()
	}
}

func (self *monitor) StopMonit() {
	self.IsNeedStop = true
	common.CheckError(errors.New("Stop process monit"), common.INFO)
}

func (self *monitor) StartMonit() {

	common.CheckError(errors.New("Start process monit"), common.INFO)
	if self.IsconfigOk == false {
		self.IsRunOk = false
		return
	}

	self.IsNeedStop = false
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		common.CheckError(err, common.ERROR)
		self.IsRunOk = false
		runtime.Goexit()
	}
	self.watcher = watcher
	if err = self.watcher.Watch(self.monitLogPath); err != nil {
		errInfo := fmt.Sprintf("%s(%s)", err.Error(), self.monitLogPath)
		common.CheckError(errors.New(errInfo), common.ERROR)
		self.IsRunOk = false
		return
	} else {
		self.doMonit()
	}
}

func (self *monitor) proGetCommand(proName string) string {
	self.sevNameMap.Mutex.Lock()
	defer self.sevNameMap.Mutex.Unlock()
	command, ok := self.sevNameMap.ConfMap[proName]
	if ok {
		return command
	}
	return "Not Found!"
}

func (self *monitor) InitConf() error {

	self._mutex_3.Lock()
	defer self._mutex_3.Unlock()

	kvIpairs := make(map[string]string, 128)
	self.monitLogPath = ""
	self.monitKeyWord = ""
	self.sendTo = ""
	for k, v := range self.ProConfMap.ConfMap {
		switch strings.ToLower(k) {
		case "monitlogpath":
			self.monitLogPath = v
		case "monitkeyword":
			self.monitKeyWord = v
		case "sendto":
			v = common.ToHeavy(v)
			self.sendTo = v
		default:
			kvIpairs[k] = v
		}
	}

	self.sevNameMap.Mutex.Lock()
	self.sevNameMap.ConfMap = kvIpairs
	self.sevNameMap.Mutex.Unlock()

	if self.monitLogPath == "" {
		errInfo := fmt.Sprintf("monitLogPath is empty, cat't monit it")
		common.CheckError(errors.New(errInfo), common.ERROR)
		self.IsconfigOk = false
	} else {
		if !common.CheckFileExcu(self.monitLogPath) {
			errInfo := fmt.Sprintf("monitLogPath(%s) is invalid, cat't monit it",
				self.monitLogPath)
			common.CheckError(errors.New(errInfo), common.ERROR)
			self.IsconfigOk = false
		} else {
			self.IsconfigOk = true
		}

	}
	if self.monitKeyWord == "" {
		self.monitKeyWord = KEYWORD
	}

	self.StopMonit()
	return nil
}

func init() {
	AlarmPro_Monitor = new(monitor)
	AlarmPro_Monitor.ProConfMap.ConfMap = make(map[string]string, 128)
	AlarmPro_Monitor.sevNameMap.ConfMap = make(map[string]string, 128)

	AlarmPro_Monitor.IsRunOk = true
	AlarmPro_Monitor.IsconfigOk = false
	AlarmPro_Monitor.IsNeedStop = false

	AlarmPro_Monitor.lastAlarmData = ""
	AlarmPro_Monitor.monitKeyWord = ""
	AlarmPro_Monitor.monitLogPath = ""
}

func getFormatName(nameString string) (string, error) {
	if len(nameString) < 3 {
		return string(""), errors.New("process name is empty!")
	}

	formatName := nameString[1 : len(nameString)-1]
	return formatName, nil
}
