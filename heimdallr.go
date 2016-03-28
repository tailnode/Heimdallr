package main

import (
	"conf"
	"container/list"
	"errors"
	"fmt"
	"log"
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

type limit struct {
	maxReqCount    int
	timeUnit       int8
	prisonTime     uint32
	prisonTimeUnit int8
}
type reqInfo struct {
	mutex           sync.Mutex
	reqTimestamp    *list.List
	prisonTimestamp int64
}
type monitor struct {
	mutex  *sync.Mutex
	info   map[uint64]*reqInfo
	ticker *time.Ticker
}

var monitors = make(map[string]monitor)
var monConfig = make(map[string]limit)
var unitMap = map[string]int8{
	"sec":  SEC,
	"min":  MIN,
	"hour": HOUR,
	"day":  DAY,
}
var exit = make(chan bool)

func newMonitor(monName string, timeUnit int8) monitor {
	log.Println("newMonitor", monName)
	m := monitor{
		mutex:  new(sync.Mutex),
		info:   make(map[uint64]*reqInfo),
		ticker: time.NewTicker(getTimeInSec(timeUnit)),
	}
	go func() {
		for {
			t := <-m.ticker.C
			log.Println(t)
			for k, v := range m.info {
				for e := v.reqTimestamp.Front(); e != nil; {
					nextElement := e.Next()
					if time.Now().Unix()-e.Value.(int64) >= int64(getTimeInSec(timeUnit)/time.Second) {
						log.Printf("remove %v from monitor[%v]id[%v]\n", e.Value.(int64), monName, k)
						v.reqTimestamp.Remove(e)
					}
					e = nextElement
				}
			}
		}
	}()
	return m
}

func getTimeInSec(unit int8) time.Duration {
	switch unit {
	case SEC:
		return time.Second
	case MIN:
		return time.Second * 60
	case HOUR:
		return time.Second * 60 * 60
	case DAY:
		return time.Second * 60 * 60 * 24
	// can't reach this branch
	default:
		return 0
	}
}

func main() {
	fmt.Println(monConfig)
	for {
		for i := 0; i < 15; i++ {
			r, err := increase("monitor1", 1)
			fmt.Printf("increase monitor1 id[1] result: %v, err:%v\n", r, err)
			r, err = increase("monitor1", 2)
			fmt.Printf("increase monitor1 id[2] result: %v, err:%v\n", r, err)
			r, err = increase("monitor2", 1)
			fmt.Printf("increase monitor2 id[1] result: %v, err:%v\n", r, err)
			r, err = increase("monitor2", 2)
			fmt.Printf("increase monitor2 id[2] result: %v, err:%v\n", r, err)
		}
		time.Sleep(time.Second * 2)
	}
	<-exit
}

func parseTimeUnit(s string) (unit int8, err error) {
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
	if _, ok := monConfig[monName]; !ok {
		return false, errors.New("invalide monitor name")
	}
	if _, ok := monitors[monName]; !ok {
		monitors[monName] = newMonitor(monName, monConfig[monName].timeUnit)
	}
	if _, ok := monitors[monName].info[id]; !ok {
		monitors[monName].mutex.Lock()
		if _, ok := monitors[monName].info[id]; !ok {
			monitors[monName].info[id] = &reqInfo{
				reqTimestamp: list.New(),
			}
		}
		monitors[monName].mutex.Unlock()
	}
	r := monitors[monName].info[id]
	r.mutex.Lock()
	if r.reqTimestamp.Len() >= monConfig[monName].maxReqCount {
		result = false
		err = errors.New("beyond limit")
	} else {
		r.reqTimestamp.PushBack(time.Now().Unix())
		result = true
	}
	r.mutex.Unlock()
	return
}

func initMonitors() {
	conf.Load("heimdallr")
	switch config := conf.GetConf("").(type) {
	case conf.Node:
		for k, item := range config {
			var maxReqCount int
			var prisonTime uint64
			var prisonTimeUnit, timeUnit int8
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
			monConfig[string(k)] = limit{
				maxReqCount:    maxReqCount,
				timeUnit:       timeUnit,
				prisonTime:     uint32(prisonTime),
				prisonTimeUnit: prisonTimeUnit,
			}
		}
	default:
		log.Println("get config failed")
	}
}
