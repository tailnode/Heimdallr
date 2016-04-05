package main

import (
	"conf"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
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
	rwMutex *sync.RWMutex
	info    map[uint64]*reqInfo
}

var monitors = make(map[string]monitor)
var monConfig map[string]limit
var exit = make(chan bool)
var loadConfRWMutex = new(sync.RWMutex)

func newMonitor(monName string) monitor {
	log.Println("newMonitor", monName)
	m := monitor{
		rwMutex: new(sync.RWMutex),
		info:    make(map[uint64]*reqInfo),
	}
	return m
}
func handler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	monName := ""
	id := uint64(0)
	paramInvalide := false

	if monNameArray, ok := req.Form["monitor_name"]; !ok {
		paramInvalide = true
	} else {
		monName = monNameArray[0]
	}
	if idArray, ok := req.Form["id"]; !ok {
		paramInvalide = true
	} else {
		err := error(nil)
		if id, err = strconv.ParseUint(idArray[0], 10, 64); err != nil {
			paramInvalide = true
		}
	}
	if paramInvalide {
		w.Write(newError(ERR_INVALIDE_PARAM, 0).toJson())
		return
	}
	result := increase(monName, id)
	w.Write(result.toJson())
}

func main() {
	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGUSR1)
	go func() {
		for {
			<-reload
			reloadConf()
		}
	}()

	http.HandleFunc("/", handler)
	port := ":" + conf.GetConf("port").(string)
	log.Fatal(http.ListenAndServe(port, nil))
	<-exit
}

// init monitor infomation from config file
func init() {
	initMonitors()
}

func reloadConf() {
	initMonitors()
}

func increase(monName string, id uint64) (err myError) {
	//  read lock for reload config file
	loadConfRWMutex.RLock()
	defer loadConfRWMutex.RUnlock()

	if _, ok := monConfig[monName]; !ok {
		return newError(ERR_MONNAME_NOT_EXIST, 0)
	}
	now := int64(time.Now().UnixNano())
	monitors[monName].rwMutex.RLock()
	r, ok := monitors[monName].info[id]
	monitors[monName].rwMutex.RUnlock()
	if !ok {
		monitors[monName].rwMutex.Lock()
		if _, ok := monitors[monName].info[id]; !ok {
			monitors[monName].info[id] = &reqInfo{
				lastReqNanoSec: now,
				mutex:          new(sync.Mutex),
			}
			r = monitors[monName].info[id]
		}
		monitors[monName].rwMutex.Unlock()
		err = newError(ERR_OK, 0)
	} else {
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
