package main

import (
	"conf"
	"container/list"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

const (
	MAX_REQ_COUNT    = "max_req_count"
	TIME_UNIT        = "time_unit"
	PRISON_TIME      = "prison_time"
	PRISON_TIME_UNIT = "prison_time_unit"
)

const (
	SEC = iota
	MIN
	HOUR
	DAY
)

var unitMap = map[string]uint8{
	"sec":  SEC,
	"min":  MIN,
	"hour": HOUR,
	"day":  DAY,
}

type limit struct {
	maxReqCount    int
	timeUnit       uint8
	prisonTime     uint32
	prisonTimeUnit uint8
}
type request struct {
	m               sync.Mutex
	reqTimestamp    *list.List
	prisonTimestamp int64
}
type monitor map[uint64]*request

var monitors = make(map[string]monitor)
var monitorLimits = make(map[string]limit)

func main() {
	fmt.Println(monitorLimits)
	for i := 0; i < 15; i++ {
		r, err := increase("monitor1", 1)
		fmt.Printf("result: %v, err:%v\n", r, err)
	}
}

func parseTimeUnit(s string) (unit uint8, err error) {
	ok := true
	if unit, ok = unitMap[s]; !ok {
		err = errors.New("invalide unit in config file")
	}
	return
}

// init monitor infomation from config file
func init() {
	initMonitors()
}

func increase(monName string, id uint64) (result bool, err error) {
	if _, ok := monitorLimits[monName]; !ok {
		return false, errors.New("invalide monitor name")
	}
	if _, ok := monitors[monName]; !ok {
		monitors[monName] = make(monitor)
	}
	if r, ok := monitors[monName][id]; !ok {
		// TODO need lock
		monitors[monName][id] = &request{
			reqTimestamp: list.New(),
		}
		monitors[monName][id].reqTimestamp.PushBack(time.Now().Unix())
		result = true
	} else {
		r.m.Lock()
		if r.reqTimestamp.Len() >= monitorLimits[monName].maxReqCount {
			result = false
			err = errors.New("beyond limit")
		} else {
			r.reqTimestamp.PushBack(time.Now().Unix())
			result = true
		}
		r.m.Unlock()
	}
	return
}

func initMonitors() {
	conf.Load("heimdallr")
	config := conf.GetConf("").(conf.Node)
	fmt.Println(config)
	for k, item := range config {
		var maxReqCount int
		var prisonTime uint64
		var timeUnit, prisonTimeUnit uint8
		var err error
		if v, ok := item.(conf.Node)[MAX_REQ_COUNT].(string); !ok {
			continue
		} else if maxReqCount, err = strconv.Atoi(v); err != nil {
			continue
		}
		if v, ok := item.(conf.Node)[TIME_UNIT].(string); !ok {
			continue
		} else if timeUnit, err = parseTimeUnit(v); err != nil {
			continue
		}
		if v, ok := item.(conf.Node)[PRISON_TIME].(string); !ok {
			continue
		} else if prisonTime, err = strconv.ParseUint(v, 10, 32); err != nil {
			continue
		}
		if v, ok := item.(conf.Node)[PRISON_TIME_UNIT].(string); !ok {
			continue
		} else if prisonTimeUnit, err = parseTimeUnit(v); err != nil {
			continue
		}
		monitorLimits[string(k)] = limit{
			maxReqCount:    maxReqCount,
			timeUnit:       timeUnit,
			prisonTime:     uint32(prisonTime),
			prisonTimeUnit: prisonTimeUnit,
		}
	}
}
