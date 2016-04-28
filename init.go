package main

import (
	"conf"
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
		for k, item := range config {
			switch item.(type) {
			case conf.Node:
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
		}
	default:
		log.Println("get config failed")
	}
}
