/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: alarmsys.go
@Version	: 5.0.20
===============================================
*/

package alarmsys

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	alarm "alarm"
	common "common"
)

//系统监控结构体
type sysMonitor struct {
	SysMonitConfMap common.Config //系统监控总的配置表
	proCpuLevel     float64       //进程cpu占用比预警阀值
	proMemValue     float64       //进程mem占用比预警阀值
	sysCpuLevel     float64       //系统和用户占用百分比总和预警阀值
	sysMemLevel     float64       //系统内存占用比预警阀值
	sysDiskLevel    float64       //系统磁盘占用比预警阀值

	SendTo       string //收件人列表
	SweepTime    int    //扫描间隔时间
	IsSysMonitOk bool   //系统监控模块是否运行正常
	IsNeedStop   bool   //是否需要停止

	_mutex_1 sync.RWMutex //针对收件人列表的读写锁
	_mutex_2 sync.RWMutex //针对进程cpu占用比预警阀值的读写锁
	_mutex_3 sync.RWMutex //针对进程mem占用比预警阀值的读写锁
	_mutex_4 sync.RWMutex //针对系统内存占用比预警阀值的读写锁
	_mutex_5 sync.RWMutex //针对系统磁盘占用比预警阀值的读写锁
	_mutex_6 sync.RWMutex //针对系统cpu占用百分比预警阀值的读写锁
}

var AlarmObject_sysMonitor *sysMonitor //单例模式

func (self *sysMonitor) sendAlarmMsg(msgInfo, subject string) {

	var node common.AlarmData
	node.To = self.SendTo
	node.Level = common.ERROR
	node.Body = "[主机IP]:\n" + common.NetworkIP + "\n\n"
	node.Body += "[告警信息]:\n"
	node.Body += msgInfo + "\n"
	node.Mailtype = common.TEXT
	node.Subject = common.SYSMO + " " + subject

	if alarm.AlarmObject_ptr.IsRunOk == true {
		alarm.AlarmObject_ptr.MsgCh <- node
	}
}

func (self *sysMonitor) StopMonit() {
	self.IsNeedStop = true
	common.CheckError(errors.New("Stop system monit"), common.INFO)
	return
}

func (self *sysMonitor) StartMonit() {
	common.CheckError(errors.New("Start system monit"), common.INFO)
	sysMemTotal := getSysMemTotal()
	if sysMemTotal == 0.0 {
		common.CheckError(errors.New("get system mem total err."), common.ERROR)
		self.IsSysMonitOk = false
		return
	}

	self.IsNeedStop = false
	for {
		if self.IsNeedStop == true {
			break
		}
		time.Sleep(time.Duration(self.SweepTime) * time.Second)

		//进程所占内存检测
		proNameList, proMemList := getProMemLevel()
		if len(proNameList) == 0 || len(proMemList) == 0 {
			common.CheckError(errors.New("get process mem level err."), common.WARN)
		} else {
			for i := 0; i < len(proMemList); i++ {
				if sysMemTotal*(proMemList[i]/100) >= self.proMemValue {
					alarmMsg := fmt.Sprintf("进程:[%s] 内存告警\n占用:%0.2fG, 阀值:%0.2fG\n",
						proNameList[i], (sysMemTotal*proMemList[i])/100, self.proMemValue)
					self.sendAlarmMsg(alarmMsg, "进程内存告警")
				}
			}
		}

		//系统所占cpu百分比
		sysCpuLevel := getSysCpuLevel()
		if sysCpuLevel == 0.0 {
			common.CheckError(errors.New("get system cpu level err."), common.WARN)
		} else {
			if sysCpuLevel > self.sysCpuLevel {
				alarmMsg := fmt.Sprintf("系统: cpu告警\n系统+用户:%0.2f%%, 阀值:%0.2f%%\n",
					sysCpuLevel, self.sysCpuLevel)
				self.sendAlarmMsg(alarmMsg, "系统CPU告警")
			}
		}

		//系统所占内存百分比
		t, u, l := getSysMemLevel()
		if t == 0.0 && u == 0.0 && l == 0.0 {
			common.CheckError(errors.New("get system mem level err."), common.WARN)
		} else {
			if l > self.sysMemLevel {
				alarmMsg := fmt.Sprintf("系统: 内存告警\nTotal: %02f M, Used: %02f M, Level: %0.2f%%, 阀值:%0.2f%%",
					t, u, l, self.sysMemLevel)
				self.sendAlarmMsg(alarmMsg, "系统内存告警")
			}
		}

		//系统所占磁盘百分比
		resultStr, diskListLevel := getSysDiskLevel()
		if len(diskListLevel) == 0 {
			common.CheckError(errors.New("get system disk level err."), common.WARN)
		} else {
			for i := 0; i < len(diskListLevel); i++ {
				if diskListLevel[i] > int(self.sysDiskLevel) {
					alarmMsg := fmt.Sprintf("系统: 磁盘(%d%%)告警 阀值: %0.2f%%\n%s\n",
						diskListLevel[i], self.sysDiskLevel, resultStr)
					self.sendAlarmMsg(alarmMsg, "系统磁盘告警")
					break
				}
			}
		}
		time.Sleep(time.Duration(self.SweepTime) * time.Second)
	}
}

func (self *sysMonitor) InitConf() error {
	self.SysMonitConfMap.Mutex.Lock()
	defer self.SysMonitConfMap.Mutex.Unlock()

	for k, v := range self.SysMonitConfMap.ConfMap {
		k_lower := strings.ToLower(k)
		switch k_lower {
		case "promemvalue":
			if v[len(v)-1:] == "G" || v[len(v)-1:] == "g" {
				v = v[:len(v)-1]
			}
			tmp, err := strconv.ParseFloat(v, 64)
			if err != nil {
				common.CheckError(err, common.ERROR)
				continue
			} else {
				self._mutex_3.Lock()
				self.proMemValue = tmp
				self._mutex_3.Unlock()
			}
		case "syscpulevel":
			if v[len(v)-1:] == "%" {
				v = v[:len(v)-1]
			}
			tmp, err := strconv.ParseFloat(v, 64)
			if err != nil {
				common.CheckError(err, common.ERROR)
				continue
			} else {
				self._mutex_6.Lock()
				self.sysCpuLevel = tmp
				self._mutex_6.Unlock()
			}
		case "sysmemlevel":
			if v[len(v)-1:] == "%" {
				v = v[:len(v)-1]
			}
			tmp, err := strconv.ParseFloat(v, 64)
			if err != nil {
				common.CheckError(err, common.ERROR)
				continue
			} else {
				self._mutex_4.Lock()
				self.sysMemLevel = tmp
				self._mutex_4.Unlock()
			}
		case "sysdisklevel":
			if v[len(v)-1:] == "%" {
				v = v[:len(v)-1]
			}
			tmp, err := strconv.ParseFloat(v, 64)
			if err != nil {
				common.CheckError(err, common.ERROR)
				continue
			} else {
				self._mutex_5.Lock()
				self.sysDiskLevel = tmp
				self._mutex_5.Unlock()
			}
		case "sweeptime":
			tmp, err := strconv.Atoi(v)
			if err != nil {
				common.CheckError(err, common.ERROR)
				continue
			} else {
				self.SweepTime = tmp
			}
		case "sendto":
			tmp := common.ToHeavy(v)
			self._mutex_1.Lock()
			self.SendTo = tmp
			self._mutex_1.Unlock()
		default:
			errInfo := fmt.Sprintf("Not Support configuration '%s', Please check ucAlarm.conf", k)
			common.CheckError(errors.New(errInfo), common.ERROR)
		}
	}

	return nil
}

func init() {
	AlarmObject_sysMonitor = new(sysMonitor)

	AlarmObject_sysMonitor.SysMonitConfMap.ConfMap = make(map[string]string, 128)
	AlarmObject_sysMonitor.proCpuLevel = 100.0
	AlarmObject_sysMonitor.proMemValue = 2.5
	AlarmObject_sysMonitor.sysMemLevel = 90.0
	AlarmObject_sysMonitor.sysDiskLevel = 90.0
	AlarmObject_sysMonitor.sysCpuLevel = 90.0

	AlarmObject_sysMonitor.SendTo = ""
	AlarmObject_sysMonitor.SweepTime = 5

	AlarmObject_sysMonitor.IsSysMonitOk = true
	AlarmObject_sysMonitor.IsNeedStop = false
}
