package main

import (
	"conf"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"strconv"
	"sync"
)

func init() {
	conf.Load("conf")
	useRedis := conf.GetConf("heimdallr/use_redis").(string)
	if useRedis == "1" {
		redisPort := conf.GetConf("heimdallr/redis_port").(string)
		redisHost := conf.GetConf("heimdallr/redis_host").(string)
		redisKey := conf.GetConf("heimdallr/redis_mon_key").(string)

		if redisPort != "" && redisHost != "" && redisKey != "" {
			initMonitorsFromRedis(redisHost, redisPort, redisKey)
		} else {
			log.Fatalln("limits.conf error, can't get redis port/host or key")
		}
	} else {
		initMonitorsFromConf()
	}
}

// init monitor infomation from redis
func initMonitorsFromRedis(host, port, key string) {
	c, err := redis.Dial("tcp", host+":"+port)
	if err != nil {
		log.Fatalf("connect to redis failed, host[%v] port[%v] err[%v]\n", host, port, err)
	} else {
		log.Printf("connect to redis success, host[%v] port[%v]\n", host, port)
	}
	defer c.Close()

	reply, err := redis.Values(c.Do("HGETALL", key))
	if err != nil {
	}
	for _, v := range reply {
		log.Println(string(v.([]byte)))
		// TODO
	}
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
