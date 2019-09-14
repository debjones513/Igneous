package main

import (
	"../../../tftp"
	"bytes"
	"net"
	"strings"
	"sync"
	"time"
)

// TFTP spec states that the data block size is fixed at 512 bytes. See item #2.

const dataBlockSize = 512

// Maps file names to file contents - TODO File size is limited, OK since this is just a code exercise

var fileCacheMap map[string]string

// Maps client addr to the last block transmitted.

var readAddrMap map[string]*RequestTracker

// Maps client addr to the last block transmitted.

var writeAddrMap map[string]*RequestTracker

// Maps client addr to the last error packet sent to the client.
// Track the timestamp so we can cleanup the list if the client fails to ack the error.
// TODO Add cleanup code.

var errorAddrMap map[string]time.Time

// Mutex to serialize metadata changes done in response to read and write requests.

var lockMetadataChanges sync.Mutex

// Mutex to serialize error map changes.

var errorMapChanges sync.Mutex


func init() {

	fileCacheMap = make(map[string]string)
	writeAddrMap = make(map[string]*RequestTracker)
	readAddrMap = make(map[string]*RequestTracker)
	errorAddrMap = make(map[string]time.Time)
}

func handleRead(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	debugLog.Printf("Handle Read Packet: %+v \n", p)

	// Take a lock while we setup and verify metadata.

	debugLog.Printf("Take Metadata Lock \n")

	lockMetadataChanges.Lock()
	defer deferredMetadataUnlock()

	// Lookup the RequestTracker object in the write request map, if it is found, return an error.
	// The RequestTracker object will be removed from this map when the write completes.
	// Do not allow reads against a file that is being written.

	if _, ok := writeAddrMap[addr.String()]; ok == true {
		sendError(pc, addr, 0, "File write is in progress.", true)
		return
	}

	// Lookup the file in our cache, return an error if the file is not found.

	if _, ok := fileCacheMap[p.Filename]; ok == false {
		sendError(pc, addr, 1, "File not found.", true)
		return
	}

	// Create a new map entry. Used to find the RequestTracker object given the client address when sending data packets.

	readAddrMap[addr.String()] = createTrackingEntry(p)

	// Spec: "RRQ ... packets are acknowledged by DATA or ERROR packets. No ack needed here,
	// just send the first data packet."

	go sendData(pc, addr, p)

	debugLog.Printf("Handle Read Packet Exit: %+v \n  %+v \n  %+v \n", fileCacheMap, readAddrMap, writeAddrMap)
}

func handleWrite(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	debugLog.Printf("Handle Write Packet: %+v \n", p)

	// Take a lock while we setup and verify metadata.

	debugLog.Printf("Take Metadata Lock \n")

	lockMetadataChanges.Lock()
	defer deferredMetadataUnlock()

	// Lookup the file in our cache, return an error if the file already exists.

	if _, ok := fileCacheMap[p.Filename]; ok == true {
		sendError(pc, addr, 1, "File already exists.", false)
		return
	}

	// Create a new cache entry for the file.

	fileCacheMap[p.Filename] = ""

	// Create a map entry. Used to find the RequestTracker object given the client address during data packet transfers.

	writeAddrMap[addr.String()] = createTrackingEntry(p)

	// Spec: "A WRQ is acknowledged with an ACK packet with block number set to zero."

	go sendAck(pc, addr, 0)

	debugLog.Printf("Handle Write Packet Exit: %+v \n  %+v \n  %+v \n", fileCacheMap, readAddrMap, writeAddrMap)
}

func handleData(pc net.PacketConn, addr net.Addr, p tftp.PacketData) {

	// If we are receiving a data packet, then the client is writing to the server.

	debugLog.Printf("Handle Data Packet: %+v \n", p)

	// Lookup the RequestTracker object, if it is not found, return an error.
	//
	// Spec: "TFTP recognizes only one error condition that does not cause
	//   termination, the source port of a received packet being incorrect.
	//   In this case, an error packet is sent to the originating host."

	if _, ok := writeAddrMap[addr.String()]; ok == false {
		sendError(pc, addr, 5, "Unknown transfer ID.", false)
		return
	}

	// Get the request tracking data.

	rt := writeAddrMap[addr.String()]

	// Serialize access to the code between Mux.Lock() and Mux.Unlock(), per client address.
	// This serializes the block writes.
	// The first block takes the lock, acks, and continues with the write.
	// A second block's request may then wait for the lock while the previous block is written.
	// Order is guaranteed.

	rt.Mux.Lock()
	defer rt.DeferredUnlock()

	debugLog.Printf("RequestTracker Lock Taken: %+v Client: %s Tracker: %+v \n", p, addr.String(), rt)

	// We ack'ed the last data packet before processing was completed, to enable better perf.
	// Check for duplicate blocks being sent, and that the last packet written corresponds to the data block
	// preceding the current data block.

	if rt.BlockNum == p.BlockNum {
		// Duplicate block resent - maybe we did not ack quick enough. Ignore it, we already wrote this block.
		// TODO Should we go ahead and ack this block a second time? I think yes, each packet should be ack'ed.
		sendAck(pc, addr, p.BlockNum)
		return
	} else if rt.BlockNum + 1 != p.BlockNum {
		sendError(pc, addr, 0, "Missing data block in transfer sequence.", false)
		return
	}

	// Send an ack to the client.
	//
	// Once the ack is sent, the client will send the next packet. We lock the meta data, so that if the next packet
	// arrives and begins processing, before processing for this packet is complete, the next packet will block
	// we can do the write and update the meta data.  This helps to eliminate any wait-time between processing
	// data packets. Note that at most 1 packet, the next packet, is waiting at any given time. Also note that
	// serializing the data packets here could result in retransmits, if the ack is not transmitted to the client
	// before the timeout expires.
	//
	// This method has the potential to eliminate all time that would be spent waiting for the next packet to be
	// transmitted over the wire - for a large number if packets, this could be a significant perf benefit.
	// On the other hand, if packets are often malformed resulting in an error applying the data,
	// then we would put the weight on successful completion rather than speed, and wait to ack.
	// Additionally, if too many packet retransmits happen, because the client timed out before getting an ack,
	// that increases net traffic, and should be considered for a final solution.
	//
	// TODO Ack tells the client we received the packet, it does not report successful processing. If we fail to write
	// TODO after ack'ing, we panic, and the server fails, or, we recover and send an error packet to terminate
	// TODO the transfer - correct?
	//
	// TODO See spec item #2, if we do not get the next expected data block, we should retransmit our ack.
	// TODO "If a packet gets lost ..."

	sendAck(pc, addr, p.BlockNum)

	// If this is the final transfer packet, and it is empty, delete the RequestTracker entry and return.
	// OK to delete here before the deferred fn to unlock runs - memory will not be garbage collected until
	// the last ref is released. The rt var takes a ref.

	if len(p.Data) == 0 {
		delete(writeAddrMap, addr.String())
		debugLog.Printf("Handle Data Packet Exit: %+v \n  %+v \n  %+v \n", fileCacheMap, readAddrMap, writeAddrMap)
		return
	}

	// Write the next block of data to the in-memory file.

	var newBlock bytes.Buffer
	newBlock.Write(p.Data)

	// TODO Next code block causes memory to hold two copies of the file data ...
	// TODO instead, use the block number to multiply by DataBlockSize and find the correct position, maybe slice
	// TODO functions...

	var fileData []string
	fileData = append(fileData, fileCacheMap[rt.PacketReq.Filename])	// Current data
	fileData = append(fileData, newBlock.String())						// New block

	fileCacheMap[rt.PacketReq.Filename] = strings.Join(fileData, "")

	// Update the meta data with the last block written and timestamp.

	rt.BlockNum = p.BlockNum
	rt.LastTranferTime = time.Now()

	// If this is the final transfer packet, delete the RequestTracker entry
	// OK to delete here before the deferred fn to unlock runs - see above.

	if len(p.Data) < dataBlockSize {
		delete(writeAddrMap, addr.String())
	}

	// TODO If the transfer for some reason stops before we receive a final transfer packet, then the file is
	// TODO partially written. Added LastTranferTime to the RequestTracker. At some point
	// TODO these should be cleaned up... and this case should not block a second transfer of the same file.
	// TODO See items #2 and #7 in the spec...

	debugLog.Printf("Handle Data Packet Exit: %+v \n  %+v \n  %+v \n", fileCacheMap, readAddrMap, writeAddrMap)
}

func handleAck(pc net.PacketConn, addr net.Addr, p tftp.PacketAck) {

	debugLog.Printf("Handle Ack Packet: %d \n", p.BlockNum)

	// Client is ack'ing an error packet.

	debugLog.Printf("Take Error Map Lock \n")

	errorMapChanges.Lock()
	defer deferredErrorMapUnlock()

	if _, ok := errorAddrMap[addr.String()]; ok == true {
		delete(errorAddrMap, addr.String())
		return
	}

	// Client is ack'ing a data packet.

	if _, ok := readAddrMap[addr.String()]; ok == false {
		sendError(pc, addr, 5, "Unknown transfer ID.", false)
		return
	}

	readAddrMap[addr.String()].Acked <- true

	debugLog.Printf("Handle Ack Packet Exit: %d \n", p.BlockNum)
}

func handleError(pc net.PacketConn, addr net.Addr, p tftp.PacketError) {

	debugLog.Printf("Handle Error Packet: %d   %s \n", p.Code, p.Msg)

	// See items #2 #7 in the spec.
}

func sendAck(pc net.PacketConn, addr net.Addr, blockNum uint16) {

	debugLog.Printf("Send Ack Packet: %+v \n", blockNum)

	// Construct an ack packet and send it to the client

	var ackPacket tftp.PacketAck
	ackPacket.BlockNum = blockNum

	b := make([]byte, 1024)
	b = ackPacket.Serialize()

	pc.WriteTo(b, addr)

	debugLog.Printf("Send Ack Packet Exit: %d \n", blockNum)
}

func sendError(pc net.PacketConn, addr net.Addr, code uint16, msg string, ackExpected bool) {

	debugLog.Printf("Send Error Packet: %+v  %d  %s \n", addr, code, msg)

	// The client will ack error packets sent during a read request.
	// The ack handler must be able to distinguish between an ack for an error packet and an ack for a data packet.

	if ackExpected {
		debugLog.Printf("Take Error Map Lock \n")

		errorMapChanges.Lock()
		defer deferredErrorMapUnlock()

		errorAddrMap[addr.String()] = time.Now()
	}

	// Construct an error packet and send it to the client

	var errorPacket tftp.PacketError
	errorPacket.Code = code
	errorPacket.Msg = msg

	b := make([]byte, 1024)
	b = errorPacket.Serialize()

	pc.WriteTo(b, addr)

	debugLog.Printf("Send Error Packet Exit: %d   %s \n", code, msg)
}

func sendData(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	debugLog.Printf("Send Data Packet: %+v \n", p)

	// Lookup the file in our cache.

	data := fileCacheMap[p.Filename]

	// The return value n is the length of the buffer; err is always nil.
	// TODO If the buffer (the file) becomes too large, Write will panic with ErrTooLarge.

	var dataBuffer bytes.Buffer
	n, _ := dataBuffer.WriteString(data)

	// Loop sending data packets until all file data has been sent.
	// Ensure that a final zero size packet is sent if needed.
	// Each data packet sent must wait for an ack from the client.
	// If a data packet gets lost, client retransmits his last ack,
	// and the server will retransmit the last packet sent.
	// Spec: "If a packet gets lost in the
	//   network, the intended recipient will timeout and may retransmit his
	//   last packet (which may be data or an acknowledgment), thus causing
	//   the sender of the lost packet to retransmit that lost packet."

	blockCount := (n / dataBlockSize) + 1
	lastBlockSize := (n % dataBlockSize)

	for i := 0;  i < blockCount; i++ {

		// Get the next block.

		start := i * dataBlockSize
		var end int

		if 	i == (blockCount - 1) {
			end = start + lastBlockSize
		} else {
			end = (i + 1) * dataBlockSize
		}

		newBlock := dataBuffer.Bytes()[start:end]

		// Construct a data packet.

		var dp tftp.PacketData
		dp.BlockNum = uint16(i + 1)				// TODO downcast is a bad idea...not production ready
		dp.Data = newBlock

		b := make([]byte, 1024)
		b = dp.Serialize()

		// Set the block number, set BlockAcked to false and start the timeout fn.

		rt := readAddrMap[addr.String()]

		rt.BlockNum = dp.BlockNum
		rt.BlockAcked = false
		rt.TimedOut = false
		go rt.TimeoutTimer()

		// Send the data packet - loop to do retries.

		for {

			pc.WriteTo(b, addr)

			debugLog.Printf("Data for get: %+v \n", b)

			go rt.RetryTimer()

			// Wait for the ack. If we get no ack within 'retry' seconds, resend.
			// We will not retransmit forever - if there is no ack, we must timeout.

			select {
			case <- rt.Acked:
				rt.BlockAcked = true
			case <- rt.Retry:
				continue
			case <- rt.Timeout:
				rt.TimedOut = true
			}

			if rt.BlockAcked {
				break
			}

			if !rt.TimedOut {
				sendError(pc, addr, 0, "Timeout", true)
				break
			}
		}

	}

	if _, ok := readAddrMap[addr.String()]; ok == true {
		delete(readAddrMap, addr.String())
	}

	debugLog.Printf("Send Data Packet Exit: %+v \n  %+v \n  %+v \n", fileCacheMap, readAddrMap, writeAddrMap)
}

func deferredMetadataUnlock() {

	lockMetadataChanges.Unlock()

	debugLog.Printf("Released Metadata Lock \n")
}

func deferredErrorMapUnlock() {

	errorMapChanges.Unlock()

	debugLog.Printf("Released Error Map Lock \n")
}

func createTrackingEntry(p tftp.PacketRequest) *RequestTracker {

	rt := new(RequestTracker)
	rt.PacketReq = p
	rt.BlockNum = 0
	rt.LastTranferTime = time.Now()
	rt.Acked = make(chan bool, 1)
	rt.Retry = make(chan bool, 1)
	rt.Timeout = make(chan bool, 1)
	rt.BlockAcked = false
	return rt
}




