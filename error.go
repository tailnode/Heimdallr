package main

import (
	"encoding/json"
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
type myErrorJson struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Extra int64  `json:"extra"`
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
func (e myError) toJson() []byte {
	eJson := myErrorJson{
		e.code,
		e.msg,
		e.extra,
	}
	b, _ := json.Marshal(eJson)
	return b
}
