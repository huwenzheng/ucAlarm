/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: alarm.go
@Version	: 5.0.20
===============================================
*/

package alarm

import (
	"errors"
	"fmt"
	"net/smtp"
	"runtime"
	"strings"
	"sync"
	"time"

	common "common"
)

type alarmObject struct {
	Host       string   //邮箱域名
	ServerAddr string   //邮箱地址信息
	User       string   //邮箱用户名
	PassWord   string   //邮箱密码
	IsRunOk    bool     //告警模块是否运行正常
	IsConfigOk bool     //是否配置成功
	IsNeedStop bool     //是否需要停止运行
	Filter     []string //发送的内容过滤

	AlarmConf common.Config         //告警模块的配置列表
	MsgCh     chan common.AlarmData //通信信道

	_mutex_1 sync.RWMutex //针对邮箱账户信息的读写锁
}

//const (
//	HOST        = "smtp.exmail.qq.com"    //默认邮箱域名
//	SERVER_ADDR = "smtp.exmail.qq.com:25" //默认邮箱地址信息
//	USER        = "uipcc_service@utry.cn" //默认邮箱用户名
//	PASSWORD    = "Utry88"                //默认邮箱密码
//)

var AlarmObject_ptr *alarmObject //单例模式

func init() {
	AlarmObject_ptr = new(alarmObject)

	AlarmObject_ptr.MsgCh = make(chan common.AlarmData, 1024)
	AlarmObject_ptr.AlarmConf.ConfMap = make(map[string]string, 128)
	AlarmObject_ptr.Filter = make([]string, 0, 128)

	AlarmObject_ptr.Host = ""
	AlarmObject_ptr.ServerAddr = ""
	AlarmObject_ptr.User = ""
	AlarmObject_ptr.PassWord = ""
	AlarmObject_ptr.IsRunOk = true
	AlarmObject_ptr.IsConfigOk = false
	AlarmObject_ptr.IsNeedStop = false
}

func (self *alarmObject) InitConf() error {

	self._mutex_1.Lock()
	defer self._mutex_1.Unlock()

	self.Filter = make([]string, 0, 128)
	self.Host = ""
	self.User = ""
	self.PassWord = ""
	self.ServerAddr = ""
	for k, v := range self.AlarmConf.ConfMap {
		switch strings.ToLower(k) {
		case "host":
			self.Host = v
		case "serveraddr":
			self.ServerAddr = v
		case "user":
			self.User = v
		case "password":
			self.PassWord = v
		default:
			if strings.Contains(strings.ToLower(k), "filter") {
				self.Filter = append(self.Filter, v)
			} else {
				errInfo := fmt.Sprintf("Not Support config '%s'", k)
				common.CheckError(errors.New(errInfo), common.ERROR)
			}
		}
	}

	if self.Host == "" || self.PassWord == "" || self.ServerAddr == "" || self.User == "" {
		self.IsConfigOk = false
		self.IsRunOk = false
		errInfo := fmt.Sprintf("Email information is wrong, Please check ucAlarm.conf")
		common.CheckError(errors.New(errInfo), common.ERROR)
	} else {
		self.IsConfigOk = true
		self.ServerAddr = self.Host + ":25"
	}
	return nil
}

func (self *alarmObject) doAlarm() {

	defer func() {
		if err := recover(); err != nil {
			common.CheckError(err, common.ERROR)
			self.IsRunOk = false
		}
	}()

	timeout := time.NewTimer(1 * time.Second)
	for {
		select {
		case msg, ok := <-self.MsgCh:
			if ok && self.IsRunOk {
				go func() {
					err := self.sendEmail(msg.To, msg.Subject, msg.Body, msg.Mailtype)
					if err != nil {
						common.CheckError(err, common.ERROR)
					} else {
						errInfo := fmt.Sprintf("<sendEmail> to: %s\ncontent: %s\n", msg.To, msg.Body)
						common.CheckError(errors.New(errInfo), common.DEBUG)
					}
				}()
			} else {
				self.IsRunOk = false
				return
			}
		case <-timeout.C:
			timeout.Reset(1 * time.Second)
			if self.IsNeedStop == true {
				return
			}
		}
	}
}

func (self *alarmObject) StartAlarm() {
	common.CheckError(errors.New("Start Alarm Module"), common.INFO)
	if self.IsConfigOk == false {
		self.IsRunOk = false
		return
	}

	self.IsNeedStop = false
	self.doAlarm()
}

func (self *alarmObject) StopAlarm() {
	self.IsNeedStop = true
	common.CheckError(errors.New("Stop Alarm Module"), common.INFO)
}

func (self *alarmObject) sendEmail(to, subject, body, mailtype string) error {
	self._mutex_1.RLock()
	defer self._mutex_1.RUnlock()

	for _, value := range self.Filter {
		if len(value) > 0 {
			if strings.Contains(body, value) {
				errInfo := fmt.Sprintf("node: %s\ncontains '%s' filtered", body, value)
				common.CheckError(errors.New(errInfo), common.DEBUG)
				return nil
			}
		} else {
			break
		}
	}
	auth := smtp.PlainAuth("", self.User, self.PassWord, self.Host)
	sendTo := strings.Split(to, ";")

	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}

	msg := []byte("To: " + to + "\r\nFrom: " + self.User + ">\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	runtime.GC()
	return smtp.SendMail(self.ServerAddr, auth, self.User, sendTo, msg)
}
