package main

import (
	"fmt"
)

const (
	ERR_OK = iota
	ERR_MONNAME_NOT_EXIST
	ERR_BEYOND_LIMIT
	ERR_IN_PRISON
	ERR_INTERAL
)

var errMap = map[int]string{
	ERR_OK:                "OK",
	ERR_MONNAME_NOT_EXIST: "monitor name not exist",
	ERR_BEYOND_LIMIT:      "beyond limit",
	ERR_IN_PRISON:         "in prison",
	ERR_INTERAL:           "internal error",
}

type myError struct {
	code  int
	msg   string
	extra int64
}

func (e myError) String() string {
	if e.code == ERR_IN_PRISON {
		return fmt.Sprintf("err code[%v] err msg[%v] prisonNanoSec[%v]", e.code, e.msg, e.extra)
	} else {
		return fmt.Sprintf("err code[%v] err msg[%v]", e.code, e.msg)
	}
}

func newError(code int, remainNanoSec int64) myError {
	if _, ok := errMap[code]; !ok {
		code = ERR_INTERAL
	}
	return myError{code, errMap[code], remainNanoSec}
}
