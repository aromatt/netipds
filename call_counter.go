package netipds

import (
	"sync"
)

type callCounter struct {
	counts map[string]uint64
	lock   sync.Mutex
}

var cc *callCounter

func DumpCallCounter() {
	cc.Dump()
}

func init() {
	cc = &callCounter{
		counts: make(map[string]uint64),
	}
}

func (cc *callCounter) Increment(fnName string) {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	if count, ok := cc.counts[fnName]; ok {
		cc.counts[fnName] = count + 1
	} else {
		cc.counts[fnName] = 1
	}
}

func (cc *callCounter) Dump() {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	for fnName, count := range cc.counts {
		println(fnName, count)
	}
}
