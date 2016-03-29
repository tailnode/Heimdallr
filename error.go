package main

import (
	"fmt"
)

const (
	ERR_OK = iota
	ERR_MONNAME_NOT_EXIST
	ERR_BEYOND_LIMIT
	ERR_INTERAL
)

var errMap = map[int]string{
	ERR_OK:                "OK",
	ERR_MONNAME_NOT_EXIST: "monitor name not exist",
	ERR_BEYOND_LIMIT:      "beyond limit",
	ERR_INTERAL:           "internal error",
}

type myError struct {
	code int
	msg  string
}

func (e myError) String() string {
	return fmt.Sprintf("err code[%v] err msg[%v]\n", e.code, e.msg)
}

func newError(code int) myError {
	if _, ok := errMap[code]; !ok {
		code = ERR_INTERAL
	}
	return myError{code, errMap[code]}
}
