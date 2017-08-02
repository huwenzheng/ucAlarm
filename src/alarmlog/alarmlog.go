/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: alarmlog.go
@Version	: 5.0.20
===============================================
*/

package alarmlog

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	alarm "alarm"
	common "common"
	uctime "uctime"
)

//日志监控结构体
type logMonitor struct {
	LogMonitorConfMap common.Config      //日志监控总体配置
	FilePathMap       common.Config      //日志路径map表
	KeyMap            common.Config      //监控关键字表
	SendTo            string             //收件人列表
	MonitedMap        map[string]logNode //正在被监控的日志文件map表
	SweepTime         int                //扫描间隔时间

	IsLogMonitorOk bool //日志监控模块是否运行正常
	IsNeedReload   bool //是否重新加载配置文件
	IsconfigOk     bool //配置是否加载正常
	IsNeedStop     bool //是否需要停止

	_mutex_2 sync.RWMutex //针对正在被监控的日志文件map表的同步锁
	_mutex_3 sync.RWMutex //针对收件人列表的同步读写锁
	_mutex_4 sync.RWMutex //针对是否重新加载配置文件的读写锁
}

//日志监控单元结构体
type logNode struct {
	srvName     string            //当前监控的服务名
	filePath    string            //当前监控的文件
	keys        string            //原始监控的keys
	MsgCh       chan int          //go程间通信管道
	IslogNodeOk bool              //监控单元是否正常
	lastMsgMap  map[string]string //最后一次关注的信息,key是关键字
	SweepTime   int               //扫描间隔时间
	logFilter   filter            //日志重复信息过滤器

	_mutex_1 sync.RWMutex //针对监控单元是否正常的同步锁
	_mutex_2 sync.RWMutex //针对最后一次关注的信息的读写同步锁
}

//重复信息过滤器
type filter struct {
	Msg      string       //重复信息
	Timer    *time.Timer  //定时器
	_mutex_1 sync.RWMutex //针对重复信息的读写锁
}

var AlarmObject_logMonitor *logMonitor //单例模式

func (self *logNode) parseLog(monitMsg string) {

	index_1 := strings.Index(monitMsg, "[")
	index_2 := strings.Index(monitMsg, "]")
	if index_1 > -1 && index_2 > -1 && index_2 > index_1+1 {
		Msg_tmp := monitMsg[index_2+1:]
		if self.logFilter.Msg != Msg_tmp {
			self.logFilter._mutex_1.Lock()
			self.logFilter.Msg = Msg_tmp
			self.logFilter._mutex_1.Unlock()
			self.logFilter.Timer.Reset(10 * time.Second)
		} else {
			self.logFilter.Timer.Reset(10 * time.Second)
			return
		}
	}

	var node common.AlarmData
	node.To = AlarmObject_logMonitor.SendTo
	node.Level = common.ERROR
	node.Body = "[主机IP]: " + common.NetworkIP + "\n"
	node.Body += "[日志文件]: " + self.filePath + "\n\n"
	node.Body += "[错误信息]:\n"
	node.Body += monitMsg
	node.Mailtype = common.TEXT
	node.Subject = common.LOGMO + " " + self.srvName + " " + common.ERROR

	errInfo := fmt.Sprintf("Send to alarm center node>\n%v", node)
	common.CheckError(errors.New(errInfo), common.DEBUG)
	if alarm.AlarmObject_ptr.IsRunOk == true {
		alarm.AlarmObject_ptr.MsgCh <- node
	}
}

func (self *logNode) proModify() error {
	for k, v := range self.lastMsgMap {
		result, err := common.DoCommand(fmt.Sprintf("cat %s | grep %s | grep %s",
			self.filePath, k, uctime.GetYMD()))
		if err != nil {
			self._mutex_1.Lock()
			self.IslogNodeOk = false
			self._mutex_1.Unlock()
			return err
		}
		if result == "" {
			result2, err2 := common.DoCommand(fmt.Sprintf("cat %s | grep %s | grep %s",
				self.filePath, k, uctime.GetHM()))
			if err2 != nil {
				self._mutex_1.Lock()
				self.IslogNodeOk = false
				self._mutex_1.Unlock()
				return err
			}
			if result2 == "" {
				continue
			}
			result = result2
		}

		resultList := strings.Split(result, "\n")
		if len(resultList) < 2 {
			return nil
		} else {
			lastResult := resultList[len(resultList)-2]
			if v == "" {
				self._mutex_2.Lock()
				self.lastMsgMap[k] = lastResult
				self._mutex_2.Unlock()
				continue
			} else if v == lastResult {
				continue
			} else {
				self._mutex_2.Lock()
				self.lastMsgMap[k] = lastResult
				self._mutex_2.Unlock()
				self.parseLog(lastResult)
			}
		}

	}

	return nil
}

func (self *logNode) doMonit_() {
	go func() {

		self.logFilter.Timer = time.NewTimer(10 * time.Second)
		for {
			if err := self.proModify(); err != nil {
				errInfo := fmt.Sprintf("monit %s err! (%s)", self.filePath, err.Error())
				common.CheckError(errors.New(errInfo), common.ERROR)
				break
			}

			select {
			case exitCode := <-self.MsgCh:
				if exitCode == 1 {
					close(self.MsgCh)
					errInfo := fmt.Sprintf("Stop monit %s", self.filePath)
					common.CheckError(errors.New(errInfo), common.INFO)
					return
				}
			case <-self.logFilter.Timer.C:
				self.logFilter._mutex_1.Lock()
				self.logFilter.Msg = ""
				self.logFilter._mutex_1.Unlock()

				self.logFilter.Timer.Reset(10 * time.Second)
			default:
				time.Sleep(time.Duration(self.SweepTime) * time.Second)
				if AlarmObject_logMonitor.IsNeedStop == true {
					return
				}
			}
		}
	}()
}

func (self *logNode) doMonit() {

	errInfo := fmt.Sprintf("Start monit log: %s", self.filePath)
	common.CheckError(errors.New(errInfo), common.DEBUG)

	AlarmObject_logMonitor._mutex_2.Lock()
	AlarmObject_logMonitor.MonitedMap[self.srvName] = *self
	AlarmObject_logMonitor._mutex_2.Unlock()

	self.doMonit_()
}

func (self *logMonitor) AddMonit(keys, srvName, fileName string) {

	if ok := common.CheckFileExcu(fileName); !ok {
		errInfo := fmt.Sprintf("Open %s failed", fileName)
		common.CheckError(errors.New(errInfo), common.ERROR)

		delete(self.FilePathMap.ConfMap, srvName)
		delete(self.KeyMap.ConfMap, srvName)
		return
	}

	var monitNode logNode

	keys_arr := strings.Split(keys, ",")
	monitNode.lastMsgMap = make(map[string]string, 0)
	monitNode.MsgCh = make(chan int, 1)
	monitNode.IslogNodeOk = true

	for _, key := range keys_arr {
		if len(key) > 2 {
			key = key[1 : len(key)-1]
			monitNode.lastMsgMap[key] = ""
		}
	}
	monitNode.srvName = srvName
	monitNode.filePath = fileName
	monitNode.keys = keys
	monitNode.SweepTime = self.SweepTime

	monitNode.doMonit()
}

func (self *logMonitor) StopMonit() {

	self._mutex_2.Lock()
	for key, value := range self.MonitedMap {
		value.MsgCh <- 1
		delete(self.MonitedMap, key)
	}
	self._mutex_2.Unlock()

	self.IsNeedStop = true
	common.CheckError(errors.New("Stop monit log"), common.INFO)
	runtime.GC()
}

func (self *logMonitor) StartMonit() {
	common.CheckError(errors.New("Start monit log"), common.INFO)
	if self.IsconfigOk == false {
		self.IsLogMonitorOk = false
		return
	}

	self.IsNeedStop = false
	for proName, file := range self.FilePathMap.ConfMap {
		if keys, ok := self.KeyMap.ConfMap[proName]; ok {
			if len(keys) <= 2 {
				continue
			}
			self.AddMonit(keys, proName, file)
		}
	}

	if len(self.MonitedMap) == 0 {
		self.IsLogMonitorOk = false
		return
	}

	for {
		self._mutex_2.Lock()
		for key, value := range self.MonitedMap {
			if value.IslogNodeOk == false {
				value.MsgCh <- 1
				delete(self.MonitedMap, key)

				self._mutex_4.Lock()
				self.IsNeedReload = true
				self._mutex_4.Unlock()
			}
		}
		self._mutex_2.Unlock()

		self._mutex_4.RLock()
		if self.IsNeedReload == true {
			for proName, keys := range self.KeyMap.ConfMap {

				node, ok := self.MonitedMap[proName]
				if ok {
					//keys 发生变动
					if keys != node.keys {
						node.MsgCh <- 1
						delete(self.MonitedMap, proName)

						fileName, ok := self.FilePathMap.ConfMap[proName]
						if ok {
							self.AddMonit(keys, proName, fileName)
						}
					}
				} else {
					//不存在
					fileName, ok := self.FilePathMap.ConfMap[proName]
					if ok {
						self.AddMonit(keys, proName, fileName)
					}
				}
			}

			for proName, fileName := range self.FilePathMap.ConfMap {
				node, ok := self.MonitedMap[proName]
				if ok {
					//文件名发生变动
					if fileName != node.filePath {
						node.MsgCh <- 1
						delete(self.MonitedMap, proName)

						keys, ok := self.KeyMap.ConfMap[proName]
						if ok {
							self.AddMonit(keys, proName, fileName)
						}
					}
				} else {
					//不存在
					keys, ok := self.KeyMap.ConfMap[proName]
					if ok {
						self.AddMonit(keys, proName, fileName)
					}
				}
			}

			for proName, node := range self.MonitedMap {
				_, ok1 := self.KeyMap.ConfMap[proName]
				_, ok2 := self.FilePathMap.ConfMap[proName]
				if !ok1 || !ok2 {
					node.MsgCh <- 1
					delete(self.MonitedMap, proName)
				}
			}
		}
		self.IsNeedReload = false
		self._mutex_4.RUnlock()

		if len(self.MonitedMap) == 0 {
			self.IsLogMonitorOk = false
			break
		}
		runtime.GC()
		time.Sleep(3 * time.Second)
	}
}

func init() {
	AlarmObject_logMonitor = new(logMonitor)
	AlarmObject_logMonitor.KeyMap.ConfMap = make(map[string]string, 128)
	AlarmObject_logMonitor.FilePathMap.ConfMap = make(map[string]string, 128)
	AlarmObject_logMonitor.LogMonitorConfMap.ConfMap = make(map[string]string, 128)
	AlarmObject_logMonitor.MonitedMap = make(map[string]logNode, 128)

	AlarmObject_logMonitor.IsLogMonitorOk = true
	AlarmObject_logMonitor.IsNeedReload = false
	AlarmObject_logMonitor.IsconfigOk = false
	AlarmObject_logMonitor.IsNeedStop = false

	AlarmObject_logMonitor.SweepTime = 3
}

func (self *logMonitor) InitConf() error {
	self.LogMonitorConfMap.Mutex.Lock()
	defer self.LogMonitorConfMap.Mutex.Unlock()

	keys := make(map[string]string, 128)
	files := make(map[string]string, 128)

	for k, v := range self.LogMonitorConfMap.ConfMap {
		k_lower := strings.ToLower(k)
		if k_lower == "sendto" {
			v = common.ToHeavy(v)
			self._mutex_3.Lock()
			self.SendTo = v
			self._mutex_3.Unlock()
		} else if k_lower == "sweeptime" {
			num, err := strconv.Atoi(v)
			if err != nil || num < 1 {
				continue
			} else {
				self.SweepTime = num
			}
		} else if strings.HasSuffix(k_lower, "key") {
			if len(v) > 2 {
				v := v[1 : len(v)-1]
				k = k[:len(k)-4]
				keys[k] = v
			}
		} else {
			files[k] = v
		}
	}

	if len(keys) != 0 && len(files) != 0 {
		self.IsconfigOk = true
	} else {
		self.IsconfigOk = false
		self.StopMonit()

		runtime.GC()
		return nil
	}

	self.KeyMap.Mutex.Lock()
	self.KeyMap.ConfMap = keys
	self.KeyMap.Mutex.Unlock()
	self.FilePathMap.Mutex.Lock()
	self.FilePathMap.ConfMap = files
	self.FilePathMap.Mutex.Unlock()

	self._mutex_4.Lock()
	self.IsNeedReload = true
	self._mutex_4.Unlock()

	runtime.GC()
	return nil
}
