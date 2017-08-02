/*
===============================================
Copyright (c) 2016 utry for ucAlarm

-----------------------------------------------
@Author : huwenzheng
@Date	: 2016/12/25
-----------------------------------------------
@FileName	: ucsig.go
@Version	: 5.0.20
===============================================
*/

package ucsig

import (
	"os"
)

type SignalHandler func(s os.Signal, arg interface{})

type SignalSet struct {
	m map[os.Signal]SignalHandler
}

func SignalSetNew() *SignalSet {
	newSig := new(SignalSet)
	newSig.m = make(map[os.Signal]SignalHandler, 32)

	return newSig
}

func (set *SignalSet) Register(s os.Signal, handler SignalHandler) {
	if _, found := set.m[s]; !found {
		set.m[s] = handler
	}
}

func (set *SignalSet) Handle(sig os.Signal, arg interface{}) {
	if _, found := set.m[sig]; found {
		set.m[sig](sig, arg)
	}
	return
}
