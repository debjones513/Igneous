package main

import (
	"../../../tftp"
	"sync"
	"time"
)

const RetryInterval = 60			// TODO Seconds to wait for an ack before resending a data packet
const TimeoutInterval = 600		// TODO Seconds to wait before timing out the transfer when retries are being sent.


// Tracks the last block sent or received per request, whether or not the request is incomplete, and the timestamp
// for the last request processed. The last field is used to cleanup stale entries.

type RequestTracker struct {
	PacketReq tftp.PacketRequest
	BlockNum uint16
	Mux sync.Mutex
	Acked chan bool					// Reads
	BlockAcked bool					// Reads
	Retry chan bool					// Reads and writes
	Timeout chan bool				// Reads and writes
	ReceivedBlockNum chan uint16	// Writes
	LastBlockWritten bool			// Writes
	PrevAckReceived chan bool		// Writes
	LastTranferTime time.Time
}

func (rt *RequestTracker) DeferredUnlock() {

	rt.Mux.Unlock()

	debugLog.Printf("Released RequestTracker Lock  %p %+v \n", &rt, rt)
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