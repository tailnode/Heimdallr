package main

import (
	"conf"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

const (
	MAX_REQ_COUNT = "max_req_count"
	TIME_UNIT     = "time_unit"
	PRISON_TIME   = "prison_time"
)

type limit struct {
	maxReqCount int
	timeUnit    uint32
	prisonTime  uint32
}
type reqInfo struct {
	mutex           sync.Mutex
	lastReqNanoSec  int64
	prisonTimestamp int64
}
type monitor struct {
	mutex *sync.Mutex
	info  map[uint64]*reqInfo
}

var monitors = make(map[string]monitor)
var monConfig = make(map[string]limit)
var exit = make(chan bool)

func newMonitor(monName string) monitor {
	log.Println("newMonitor", monName)
	m := monitor{
		mutex: new(sync.Mutex),
		info:  make(map[uint64]*reqInfo),
	}
	return m
}

func main() {
	fmt.Println(monConfig)
	for {
		for i := 0; i < 15; i++ {
			r, err := increase("monitor1", 1)
			fmt.Printf("increase monitor1 id[1] %v result: %v, err:%v\n", i, r, err)
		}
		for i := 0; i < 15; i++ {
			r, err := increase("monitor1", 1)
			fmt.Printf("increase monitor1 id[1] %v result: %v, err:%v\n", i, r, err)
			time.Sleep(time.Millisecond * 100)
		}
	}
	<-exit
}

// init monitor infomation from config file
func init() {
	initMonitors()
}

func increase(monName string, id uint64) (result bool, err error) {
	if _, ok := monConfig[monName]; !ok {
		return false, errors.New("invalide monitor name")
	}
	if _, ok := monitors[monName]; !ok {
		monitors[monName] = newMonitor(monName)
	}
	now := int64(time.Now().UnixNano())
	if _, ok := monitors[monName].info[id]; !ok {
		monitors[monName].mutex.Lock()
		if _, ok := monitors[monName].info[id]; !ok {
			monitors[monName].info[id] = &reqInfo{
				lastReqNanoSec: now,
			}
		}
		monitors[monName].mutex.Unlock()
	} else {
		r := monitors[monName].info[id]
		r.mutex.Lock()
		if (now-r.lastReqNanoSec)*int64(monConfig[monName].maxReqCount) < int64(monConfig[monName].timeUnit)*int64(time.Second) {
			result = false
			err = errors.New("beyond limit")
		} else {
			r.lastReqNanoSec = now
			result = true
		}
		r.mutex.Unlock()
	}
	return
}

func initMonitors() {
	conf.Load("heimdallr")
	switch config := conf.GetConf("").(type) {
	case conf.Node:
		for k, item := range config {
			var maxReqCount int
			var prisonTime uint64
			var timeUnit uint64
			var err error
			if v, ok := item.(conf.Node)[MAX_REQ_COUNT].(string); !ok {
				continue
			} else if maxReqCount, err = strconv.Atoi(v); err != nil {
				continue
			}
			if v, ok := item.(conf.Node)[TIME_UNIT].(string); !ok {
				continue
			} else if timeUnit, err = strconv.ParseUint(v, 10, 32); err != nil {
				continue
			}
			if v, ok := item.(conf.Node)[PRISON_TIME].(string); !ok {
				continue
			} else if prisonTime, err = strconv.ParseUint(v, 10, 32); err != nil {
				continue
			}
			monConfig[string(k)] = limit{
				maxReqCount: maxReqCount,
				timeUnit:    uint32(timeUnit),
				prisonTime:  uint32(prisonTime),
			}
		}
	default:
		log.Println("get config failed")
	}
}
