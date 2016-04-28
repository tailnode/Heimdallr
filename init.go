package main

import (
	"conf"
	"fmt"
	"log"
	"strconv"
	"sync"
)

func init() {
	conf.Load("conf")
	initMonitorsFromConf()
}

// init monitor infomation from redis
func initMonitorsFromRedis() {
}

// init monitor infomation from config file
func initMonitorsFromConf() {
	switch config := conf.GetConf("limits").(type) {
	case conf.Node:
		loadConfRWMutex.Lock()
		defer loadConfRWMutex.Unlock()
		monConfig = make(map[string]limit) // get new config
		for k, itemRaw := range config {
			switch item := itemRaw.(type) {
			case conf.Node:
				var maxReqCount int
				var timeUnit uint64
				var prisonTime uint64
				var err error
				if v, ok := item[MAX_REQ_COUNT].(string); !ok {
					continue
				} else if maxReqCount, err = strconv.Atoi(v); err != nil {
					continue
				} else if maxReqCount <= 0 {
					continue
				}
				if v, ok := item[TIME_UNIT].(string); !ok {
					continue
				} else if timeUnit, err = strconv.ParseUint(v, 10, 32); err != nil {
					continue
				}
				if v, ok := item[PRISON_TIME].(string); !ok {
					continue
				} else if prisonTime, err = strconv.ParseUint(v, 10, 32); err != nil {
					continue
				}
				monName := string(k)
				monConfig[monName] = limit{
					maxReqCount: maxReqCount,
					timeUnit:    uint32(timeUnit),
					prisonTime:  uint32(prisonTime),
					mutex:       new(sync.Mutex),
				}
				if _, ok := monitors[monName]; !ok {
					monitors[monName] = newMonitor(monName)
				}
			}
		}
	default:
		log.Println("get config failed")
	}
	fmt.Println(monConfig)
}
