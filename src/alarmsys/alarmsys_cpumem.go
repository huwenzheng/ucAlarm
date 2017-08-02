/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: alarmsys_cpumem.go
@Version	: 5.0.20
===============================================
*/

package alarmsys

import (
	"bytes"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	common "common"
)

//系统内存总量
func getSysMemTotal() float64 {
	result, err := common.DoCommand("free -m")
	if err != nil {
		common.CheckError(err, common.ERROR)
		return 0.0
	}
	reg := regexp.MustCompile(`[\d]+`)
	data := reg.FindString(result)
	total, err := strconv.Atoi(data)
	if err != nil {
		common.CheckError(err, common.ERROR)
		return 0.0
	}
	return float64(total) / 1024
}

//系统所占磁盘百分比
func getSysDiskLevel() (result string, diskLevelList []int) {
	result, err := common.DoCommand("df -h")
	if err != nil {
		common.CheckError(err, common.ERROR)
		return
	}
	dataList := strings.Split(result, "\n")

	reg := regexp.MustCompile(`[\d]+%`)
	for i := 1; i < len(dataList)-1; i++ {
		data := reg.FindString(dataList[i])
		tmp := data[:len(data)-1]
		dataValue, err := strconv.Atoi(tmp)
		if err != nil {
			common.CheckError(err, common.ERROR)
			break
		}
		diskLevelList = append(diskLevelList, dataValue)
	}
	return
}

//系统所占内存百分比
func getSysMemLevel() (float64, float64, float64) {
	result, err := common.DoCommand("free -m")
	if err != nil {
		common.CheckError(err, common.ERROR)
		return 0.0, 0.0, 0.0
	}
	reg := regexp.MustCompile(`[\d]+`)
	dataList := reg.FindAllString(result, -1)
	if len(dataList) < 2 {
		return 0.0, 0.0, 0.0
	} else {
		total, err := strconv.Atoi(dataList[0])
		if err != nil {
			common.CheckError(err, common.ERROR)
			return 0.0, 0.0, 0.0
		}
		used, err := strconv.Atoi(dataList[1])
		if err != nil {
			common.CheckError(err, common.ERROR)
			return 0.0, 0.0, 0.0
		}
		return float64(total), float64(used), (float64(used) / float64(total)) * 100
	}
}

//系统所占cpu百分比
func getSysCpuLevel() float64 {
	result, err := common.DoCommand2("top -bn 2 -d 1 | grep Cpu")
	if err != nil {
		common.CheckError(err, common.ERROR)
		return 0.0
	}

	cpuDataList := strings.Split(result, "\n")
	reg := regexp.MustCompile(`[\d]+.[\d]+`)
	cpudataList := reg.FindAllString(cpuDataList[1], -1)
	if len(cpudataList) < 2 {
		return 0.0
	} else {
		cpuUser, err := strconv.ParseFloat(cpudataList[0], 64)
		if err != nil {
			common.CheckError(err, common.ERROR)
			return 0.0
		}
		cpuSys, err := strconv.ParseFloat(cpudataList[1], 64)
		if err != nil {
			common.CheckError(err, common.ERROR)
			return 0.0
		}
		return cpuUser + cpuSys
	}
}

//进程所占内存报警阀值
func getProMemLevel() (proName []string, memList []float64) {
	cmd := exec.Command("ps", "aux")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		common.CheckError(err, common.ERROR)
		return
	}

	for {
		line, err := out.ReadString('\n')
		if err != nil {
			break
		}

		tokens := strings.Split(line, " ")
		ft := make([]string, 0)
		for _, t := range tokens {
			if t != "" && t != "\t" {
				ft = append(ft, t)
			}
		}

		mem, err := strconv.ParseFloat(ft[3], 64)
		if err != nil {
			continue
		}
		proName = append(proName, ft[10])
		memList = append(memList, mem)
	}
	return
}
