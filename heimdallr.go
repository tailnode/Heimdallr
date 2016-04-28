package main

import (
	"conf"
	"fmt"
	"log"
	"net/http"
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
		w.Write([]byte("invalide parameter"))
		return
	}
	result := increase(monName, id)
	w.Write([]byte(result.String()))
}

func main() {
	fmt.Println(monConfig)

	http.HandleFunc("/", handler)
	port := ":" + conf.GetConf("heimdallr/heimdallr_http_port").(string)
	log.Fatal(http.ListenAndServe(port, nil))
	<-exit
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
		err = newError(ERR_OK, 0)
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
