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
// for the last request processed. The last two fields are used to cleanup stale entries.

type RequestTracker struct {
	PacketReq tftp.PacketRequest
	BlockNum uint16
	Mux sync.Mutex
	Acked chan bool
	Retry chan bool
	Timeout chan bool
	BlockAcked bool
	TransferIncomplete bool
	LastTranferTime time.Time
}

func (rt *RequestTracker) DeferredUnlock() {

	fmt.Printf("Releasing RequestTracker Lock %+v \n", rt)

	rt.Mux.Unlock()
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