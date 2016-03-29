package main

import (
	"fmt"
	"testing"
	"time"
)

func increaseWraper(monName string, id uint64, c chan int) {
	err := increase(monName, id)
	c <- err.code
}

// config file illegal
func TestConfigErr(t *testing.T) {
}

// monitor name not exist
func TestMonNameNotExist(t *testing.T) {
	fmt.Println("start TestMonNameNotExist")
	id := uint64(0)
	if err := increase("notexist", id); err.code != ERR_MONNAME_NOT_EXIST {
		t.Errorf("TestMonNameNotExist failed, want[%s] get[%s]\n", ERR_MONNAME_NOT_EXIST, err.code)
	}
}

func TestAllAllow(t *testing.T) {
	fmt.Println("start TestAllAllow")
	id := uint64(1)
	for i := 0; i < 15; i++ {
		if err := increase("monitor1", id); err.code != ERR_OK {
			t.Errorf("increase monitor1 id[%v] %v result: %v", id, i, err)
		}
		time.Sleep(time.Millisecond * 100)
	}
}
func TestAllReject(t *testing.T) {
	fmt.Println("start TestAllReject")
	id := uint64(2)
	increase("monitor1", id)
	for i := 0; i < 9; i++ {
		time.Sleep(time.Millisecond * 9)
		if err := increase("monitor1", id); err.code != ERR_BEYOND_LIMIT {
			t.Errorf("increase monitor1 id[%v] %v result: %v", id, i, err)
		}
	}
}
func TestiAllowReject(t *testing.T) {
	fmt.Println("start TestiAllowReject")
	id := uint64(3)
	increase("monitor1", id)
	for i := 0; i < 9; i++ {
		time.Sleep(time.Millisecond * 90)
		err := increase("monitor1", id)
		if i%2 == 0 {
			if err.code != ERR_BEYOND_LIMIT {
				t.Errorf("increase monitor1 id[%v] %v result: %v", id, i, err)
			}
		} else {
			if err.code != ERR_OK {
				t.Errorf("increase monitor1 id[%v] %v result: %v", id, i, err)
			}
		}
	}
}
func TestParallel(t *testing.T) {
	fmt.Println("start TestParallel")
	id := uint64(0)
	monName := "monitor2"
	const CHAN_NUM = 20
	var chans [CHAN_NUM]chan int
	for i := 0; i < CHAN_NUM; i++ {
		chans[i] = make(chan int)
	}
	for i := 0; i < len(chans); i++ {
		go increaseWraper(monName, id, chans[i])
	}
	sumOk := 0
	for i := 0; i < len(chans); i++ {
		if <-chans[i] == ERR_OK {
			sumOk++
		}
	}
	if sumOk != 1 {
		t.Errorf("TestParallel failed, should allow %v request, but there are %v\n", 1, sumOk)
	}
}
func TestPrison1(t *testing.T) {
	fmt.Println("start TestPrison1")
	id := uint64(0)
	monName := "monitor3"
	increase(monName, id) // ok
	increase(monName, id) // beyond limit, put in prison
	for i := 0; i < 10; i++ {
		// in prison
		if err := increase(monName, id); err.code != ERR_IN_PRISON {
			t.Errorf("TestPrison1 failed, want [%v] get[%v]\n", ERR_IN_PRISON, err.code)
		}
	}
	time.Sleep(time.Second * 3)
	// prison time enough, free it
	if err := increase(monName, id); err.code != ERR_OK {
		t.Errorf("TestPrison1 failed, want [%v] get[%v]\n", ERR_IN_PRISON, err.code)
	}
}
func TestPrison2(t *testing.T) {
	fmt.Println("start TestPrison2")
	id := uint64(0)
	monName := "monitor4"
	increase(monName, id) // ok
	increase(monName, id) // beyond limit, put in prison
	for i := 0; i < 10; i++ {
		// in prison
		if err := increase(monName, id); err.code != ERR_IN_PRISON {
			t.Errorf("TestPrison2 failed, want [%v] get[%v]\n", ERR_IN_PRISON, err.code)
		}
	}
	time.Sleep(time.Second)
	// prison time enough, free it, but reentry prison
	if err := increase(monName, id); err.code != ERR_BEYOND_LIMIT {
		t.Errorf("TestPrison2 failed, want [%v] get[%v]\n", ERR_BEYOND_LIMIT, err.code)
	}
	time.Sleep(time.Second * 2)
	// prison time enough, free it
	if err := increase(monName, id); err.code != ERR_OK {
		t.Errorf("TestPrison2 failed, want [%v] get[%v]\n", ERR_OK, err.code)
	}
}
