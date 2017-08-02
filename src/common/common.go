/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: common.go
@Version	: 5.0.20
===============================================
*/

package common

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	uctime "uctime"
)

var errFile *os.File   //错误日志文件句柄
var DebugFile *os.File //debug 日志文件句柄
var WorkPath string    //系统运行路径

//邮件告警数据
type AlarmData struct {
	To       string //发送给谁
	Subject  string //邮件主题
	Mailtype string //邮件文本类型
	Body     string //具体消息体
	Level    string //消息级别
}

//配置数据
type Config struct {
	ConfMap map[string]string //放置配置的map表
	Mutex   sync.RWMutex      //同步锁
}

//告警级别
const (
	DEBUG = "Debug"   //调试级别
	INFO  = "Info"    //信息级别
	WARN  = "Warning" //警告级别
	ERROR = "Error"   //错误级别
	FATAL = "Fatal"   //致命级别
)

//服务类型
const (
	PRORE = "[进程重启]"
	LOGMO = "[日志监控]"
	SYSMO = "[系统监控]"
	BUSMO = "[业务监控]"
	INTRE = "[接口查询]"
)

//文本格式
const (
	TEXT = "text"
	HTML = "html"
)

var (
	Pversion        bool   //告警系统版本号
	UcAlarmConfPath string //告警系统配置文件路径
	IsReloadOpt     bool   //是否配置文件重载操作
	IsDebug         bool   //debug 模式
	NetworkIP       string //本地ip地址
)

//执行系统命令
func DoCommand(commandStr string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", commandStr)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		if err.Error() == "exit status 1" {
			return string(""), nil
		}
		return string(""), err
	}
	return out.String(), nil
}

//执行动态系统命令
func DoCommand2(commandStr string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", commandStr)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		if err.Error() == "exit status 1" {
			return string(out.Bytes()), nil
		}
		return string(""), err
	}
	return out.String(), nil
}

//收件人去重
func ToHeavy(str string) string {
	if len(str) == 0 {
		return str
	}

	var ret []string
	to := strings.Split(str, ";")
	sort.Strings(to)

	to_len := len(to)
	for i := 0; i < to_len; i++ {
		if i > 0 && to[i-1] == to[i] {
			continue
		}
		ret = append(ret, to[i])
	}
	return strings.Join(ret, ";")
}

//内容过滤
func ProImagFilter(images *string, proName string) string {
	imageList := strings.Split(*images, "\n")

	var needImags string
	for _, value := range imageList {
		if strings.Contains(value, proName) {
			needImags += (value + "\n")
		}
	}
	return needImags
}

//获取下一秒
func getNextSecond(nowTime string) (string, error) {
	var h, m, s int
	times := strings.Split(nowTime, ":")
	if len(times) != 3 {
		return string(""), errors.New("time format err!")
	}
	h, _ = strconv.Atoi(times[0])
	m, _ = strconv.Atoi(times[1])
	s, _ = strconv.Atoi(times[2])

	s++
	if s > 59 {
		s = 0
		m++
		if m > 59 {
			m = 0
			h++
			if h > 23 {
				h = 0
			}
		}
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s), nil
}

//获取本地ip地址
func GetNetworkIp() error {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err
	}

	for _, address := range addrs {

		//检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {

				localIp := ipnet.IP.String()
				//检查是否为容器地址
				if strings.Contains(localIp, "172") {
					continue
				} else {
					NetworkIP = localIp
					break
				}
			}
		}
	}

	if len(NetworkIP) < 7 {
		return errors.New("Get Network Ip err!")
	}
	return nil
}

//检查文件是否存在
func CheckFileExcu(fileName string) bool {
	fs, err := os.Open(fileName)
	if err != nil {
		return false
	}

	defer fs.Close()
	return true
}

//获取文件名
func GetFileName() string {
	_, file, _, ok := runtime.Caller(2)
	if ok {
		return file
	}

	return ""
}

//获取代码执行行数
func GetFileLine() int {
	_, _, line, ok := runtime.Caller(2)
	if ok {
		return line
	}

	return -1
}

//错误检查机制
func CheckError(err interface{}, types string) {
	if err != nil && types != "" {

		msgInfo := fmt.Sprintf("[%s][ucAlarm]==>>[%s:%d][%s]: %v\n",
			uctime.GetFormatDate(), GetFileName(), GetFileLine(), types, err)
		if types == "Error" {
			errFile.WriteString(msgInfo)
		} else if types == "Debug" {
			if IsDebug {
				DebugFile.WriteString(msgInfo)
			}
		}
		//fmt.Printf(msgInfo)
	}
}

func init() {
	errFile, _ = os.Create("logs/ucAlarm.error")

	WorkPath, _ = DoCommand("pwd")
	if WorkPath[len(WorkPath)-1:] == "\n" {
		WorkPath = WorkPath[:len(WorkPath)-1]
	}
}
