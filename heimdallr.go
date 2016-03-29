package main

import (
	"conf"
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
	mutex       *sync.Mutex
}
type reqInfo struct {
	lastReqNanoSec  int64
	inPrisonNanoSec int64
	mutex           *sync.Mutex
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
	<-exit
}

// init monitor infomation from config file
func init() {
	initMonitors()
}

func increase(monName string, id uint64) (err myError) {
	if _, ok := monConfig[monName]; !ok {
		return newError(ERR_MONNAME_NOT_EXIST, 0)
	}
	if _, ok := monitors[monName]; !ok {
		monConfig[monName].mutex.Lock()
		if _, ok := monitors[monName]; !ok {
			monitors[monName] = newMonitor(monName)
		}
		monConfig[monName].mutex.Unlock()
	}
	now := int64(time.Now().UnixNano())
	if _, ok := monitors[monName].info[id]; !ok {
		monitors[monName].mutex.Lock()
		if _, ok := monitors[monName].info[id]; !ok {
			monitors[monName].info[id] = &reqInfo{
				lastReqNanoSec: now,
				mutex:          new(sync.Mutex),
			}
		}
		monitors[monName].mutex.Unlock()
	} else {
		r := monitors[monName].info[id]
		r.mutex.Lock()
		defer r.mutex.Unlock()
		//log.Println(now, r.lastReqNanoSec, monConfig[monName].maxReqCount, monConfig[monName].timeUnit, int64(time.Second))
		// in prison
		if r.inPrisonNanoSec != 0 && monConfig[monName].prisonTime != 0 {
			// stay in prison time enough, free it
			if now-r.inPrisonNanoSec > int64(monConfig[monName].prisonTime)*int64(time.Second) {
				r.inPrisonNanoSec = 0
			} else {
				//stay in prison continue
				err = newError(ERR_IN_PRISON, int64(monConfig[monName].prisonTime)*int64(time.Second)-now+r.inPrisonNanoSec)
				return
			}
		}
		// check if beyond limit
		if (now-r.lastReqNanoSec)*int64(monConfig[monName].maxReqCount) < int64(monConfig[monName].timeUnit)*int64(time.Second) {
			err = newError(ERR_BEYOND_LIMIT, 0)
			if monConfig[monName].prisonTime != 0 {
				// put in prison
				r.inPrisonNanoSec = now
			}
		} else {
			r.lastReqNanoSec = now
			err = newError(ERR_OK, 0)
		}
	}
	return
}

func initMonitors() {
	conf.Load("heimdallr")
	switch config := conf.GetConf("").(type) {
	case conf.Node:
		for k, item := range config {
			var maxReqCount int
			var timeUnit uint64
			var prisonTime uint64
			var err error
			if v, ok := item.(conf.Node)[MAX_REQ_COUNT].(string); !ok {
				continue
			} else if maxReqCount, err = strconv.Atoi(v); err != nil {
				continue
			} else if maxReqCount <= 0 {
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
				mutex:       new(sync.Mutex),
			}
		}
	default:
		log.Println("get config failed")
	}
}
