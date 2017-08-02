/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: uctime.go
@Version	: 5.0.20
===============================================
*/

package uctime

import (
	"fmt"
	"time"
)

var months = [...]string{
	"January",
	"February",
	"March",
	"April",
	"May",
	"June",
	"July",
	"August",
	"September",
	"October",
	"November",
	"December",
}

func GetWeek() string {
	t := time.Now()
	return t.Weekday().String()
}

func GetDay() int {
	t := time.Now()
	return t.Day()
}

func GetMonth_str() string {
	t := time.Now()
	month_str := t.Month().String()
	month_str_3 := month_str[:2]

	return month_str_3
}

func GetMonth() int {
	t := time.Now()
	month_str := t.Month().String()
	for n, value := range months {
		if month_str == value {
			return n + 1
		}
	}
	return -1
}

func GetYear() int {
	t := time.Now()
	return t.Year()
}

func GetHM() string {
	t := time.Now()
	hour := t.Hour()
	minu := t.Minute()

	hm := fmt.Sprintf("%02d:%02d", hour, minu)
	return hm[:len(hm)-1]
}

func GetFormatDate() string {
	t := time.Now()
	year := t.Year()
	month := t.Month().String()
	day := t.Day()
	hour := t.Hour()
	minu := t.Minute()
	second := t.Second()

	for n, value := range months {
		if month == value {
			return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
				year, n+1, day, hour, minu, second)
		}
	}

	return "2016-01-01 00:00:00"
}

func GetYMD() string {
	return fmt.Sprintf("%04d-%02d-%02d", GetYear(), GetMonth(), GetDay())
}

func GetHour() int {
	t := time.Now()
	return t.Hour()
}

func GetStdDate() string {
	t := time.Now()
	return t.String()
}
