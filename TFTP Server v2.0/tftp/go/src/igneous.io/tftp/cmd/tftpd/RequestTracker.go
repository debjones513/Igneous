package main

import (
	"../../../tftp"
	"fmt"
	"sync"
	"time"
)

const RetryInterval = 3			// TODO Seconds to wait for an ack before resending a data packet
const TimeoutInterval = 15		// TODO Seconds to wait before timing out the transfer when retries are being sent.


// Tracks the last block sent or received per request, whether or not the request is incomplete, and the timestamp
// for the last request processed. The last field is used to cleanup stale entries.

type RequestTracker struct {
	PacketReq tftp.PacketRequest
	BlockNum uint16
	Mux sync.Mutex
	Acked chan bool
	Retry chan bool
	Timeout chan bool
	BlockAcked bool
	TimedOut bool
	LastTranferTime time.Time
}

func (rt *RequestTracker) DeferredUnlock() {

	rt.Mux.Unlock()

	fmt.Printf("Released RequestTracker Lock  %p %+v \n", &rt, rt)
}

func (rt *RequestTracker) RetryTimer()  {
	time.Sleep(time.Second * RetryInterval)
	rt.Timeout <- true
	return
}

func (rt *RequestTracker) TimeoutTimer()  {
	time.Sleep(time.Second * TimeoutInterval)
	rt.Timeout <- true
	return
}