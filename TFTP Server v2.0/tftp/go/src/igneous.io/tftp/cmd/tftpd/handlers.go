package main

import (
	"../../../tftp"
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// TFTP spec states that the data block size is fixed at 512 bytes. See tem #2.

const dataBlockSize = 512

// Maps file names to file contents - TODO File size is limited, OK since this is just a code exercise

var fileCacheMap map[string]string

// Maps client addr to the last block transmitted.

var readAddrMap map[string]*RequestTracker

// Maps client addr to the last block transmitted.

var writeAddrMap map[string]*RequestTracker

// Mutex to serialize metadata changes done in response to read and write requests.

var lockMetadataChanges sync.Mutex

// Globals to enum lists

var files *map[string]string
var read_addrs *map[string]*RequestTracker
var write_addrs *map[string]*RequestTracker

func init() {

	fileCacheMap = make(map[string]string)
	writeAddrMap = make(map[string]*RequestTracker)
	readAddrMap = make(map[string]*RequestTracker)

	files = &fileCacheMap
	read_addrs = &readAddrMap
	write_addrs = &writeAddrMap
}

func handleRead(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Read Packet: %+v \n", p)

	// Take a lock while we setup and verify metadata.

	lockMetadataChanges.Lock()
	defer deferredMetadataUnlock()

	// Lookup the RequestTracker object in the write request map, if it is found, return an error.
	// The RequestTracker object will be removed from this map when the write completes.
	// Do not alow reads against a file that is being written.

	if _, ok := writeAddrMap[addr.String()]; ok == true {
		sendError(pc, addr, 0, "File write is in progress.")
		return
	}

	// Lookup the file in our cache, return an error if the file is not found.

	if _, ok := fileCacheMap[p.Filename]; ok == false {
		sendError(pc, addr, 1, "File not found.")
	}

	// Create a map entry to find the RequestTracker object given the client address when sending data packets.

	readAddrMap[addr.String()] = createTrackingEntry(p)

	// Spec: "RRQ ... packets are acknowledged by DATA or ERROR packets. No ack needed here,
	// just send the first data packet."

	go sendData(pc, addr, p)
}

func handleWrite(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Write Packet: %+v \n", p)

	// Take a lock while we setup and verify metadata.

	lockMetadataChanges.Lock()
	defer deferredMetadataUnlock()

	// Lookup the file in our cache, return an error if the file already exists.

	if _, ok := fileCacheMap[p.Filename]; ok == true {
		sendError(pc, addr, 1, "File already exists.")
		return
	}

	// Create a new cache entry for the file.

	fileCacheMap[p.Filename] = ""

	// Create a map entry. Used to find the RequestTracker object given the client address during data packet transfers.

	writeAddrMap[addr.String()] = createTrackingEntry(p)

	// Spec: "A WRQ is acknowledged with an ACK packet with block number set to zero."

	go sendAck(pc, addr, 0)

	fmt.Printf("Handle Write Packet: %+v \n  %+v \n  %+v \n", files, read_addrs, write_addrs)
}

func handleData(pc net.PacketConn, addr net.Addr, p tftp.PacketData) {

	// If we are receiving a data packet, then the client is writing to the server.

	fmt.Printf("Handle Data Packet: %+v \n", p)

	// Lookup the RequestTracker object, if it is not found, return an error.
	//
	// Spec: "TFTP recognizes only one error condition that does not cause
	//   termination, the source port of a received packet being incorrect.
	//   In this case, an error packet is sent to the originating host."

	if _, ok := writeAddrMap[addr.String()]; ok == false {
		sendError(pc, addr, 5, "Unknown transfer ID.")
		return
	}

	// Get the current tracking data.

	rt := writeAddrMap[addr.String()]

	// Serialize access to the code between Mux.Lock() and Mux.Unlock(), per client address.

	rt.Mux.Lock()
	defer rt.DeferredUnlock()

	fmt.Printf("Data Packet Lock Taken: %+v Client: %s Tracker: %+v \n", p, addr.String(), rt)

	// We ack'ed the last data packet before processing was completed, to enable better perf.
	// Check for duplicate blocks being sent, and that the last packet written corresponds to the data block
	// preceding the current data block.

	if rt.BlockNum == p.BlockNum {
		// Duplicate block resent - maybe we did not ack quick enough. Ignore it, we already wrote this block.
		// TODO Should we go ahead and ack this block a second time?
		sendAck(pc, addr, p.BlockNum)
		return
	} else if rt.BlockNum + 1 != p.BlockNum {
		sendError(pc, addr, 0, "Missing data block in transfer sequence.")
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

	// If this is the final transfer packet, and it is empty, delete the Tracker entry and return.

	if len(p.Data) == 0 {
		delete(writeAddrMap, addr.String())
		return
	}

	// Write the next block of data to the in-memory file.

	var new_block bytes.Buffer
	new_block.Write(p.Data)

	// TODO memory now holds two copies of the file data ...
	// TODO instead, use the block number to multiply by DataBlockSize and find the correct position, maybe slice functions...

	var file_data []string
	file_data = append(file_data, fileCacheMap[rt.PacketReq.Filename])	// Current data
	file_data = append(file_data, new_block.String())					// New block

	fileCacheMap[rt.PacketReq.Filename] = strings.Join(file_data, "")

	// Update the meta data with the last block written and timestamp.

	rt.BlockNum = p.BlockNum
	rt.LastTranferTime = time.Now()

	// If this is the final transfer packet, delete the RequestTracker entry

	if len(p.Data) < dataBlockSize {
		delete(writeAddrMap, addr.String())
	}

	// TODO If the transfer for some reason stops before we receive a final transfer packet, then the file is
	// TODO partially written. Added a bit to the RequestTracker to signify incomplete transfer. At some point
	// TODO these should be cleaned up... and this case should not block a second transfer of the same file.
	// TODO See items #2 and #7 in the spec...

	fmt.Printf("Handle Write Packet: %+v \n  %+v \n  %+v \n", files, read_addrs, write_addrs)
}

func handleAck(pc net.PacketConn, addr net.Addr, p tftp.PacketAck) {

	fmt.Printf("Handle Ack Packet: %+v \n", p)

	// If we received an ack, it is in response to a data packet being sent.
	// Update the RequestTracking data.
	// TODO use a channel to signal the send_data fn to send the next block.

	if _, ok := readAddrMap[addr.String()]; ok == false {
		sendError(pc, addr, 5, "Unknown transfer ID.")
		return
	}

	readAddrMap[addr.String()].Acked <- true

}

func handleError(pc net.PacketConn, addr net.Addr, p tftp.PacketError) {

	fmt.Printf("Handle Error Packet Packet: %+v \n", p)

	// TODO See item #7 in the spec...
}

func sendAck(pc net.PacketConn, addr net.Addr, block_num uint16) {

	fmt.Printf("Send Ack Packet: %+v \n", block_num)

	// Construct an ack packet and send it to the client

	var ack_packet tftp.PacketAck
	ack_packet.BlockNum = block_num

	b := make([]byte, 1024)
	b = ack_packet.Serialize()

	pc.WriteTo(b, addr)
}

func sendError(pc net.PacketConn, addr net.Addr, Code uint16, Msg  string) {

	fmt.Printf("Handle Error Packet: %+v \n", addr)

	// Construct an error packet and send it to the client

	var error_packet tftp.PacketError
	error_packet.Code = Code
	error_packet.Msg = Msg

	b := make([]byte, 1024)
	b = error_packet.Serialize()

	pc.WriteTo(b, addr)
}

func sendData(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Read Packet Packet: %+v \n", p)

	// Lookup the file in our cache.

	data := fileCacheMap[p.Filename]

	// The return value n is the length of the buffer; err is always nil.
	// TODO If the buffer (the file) becomes too large, Write will panic with ErrTooLarge.

	var data_buffer bytes.Buffer
	n, _ := data_buffer.WriteString(data)

	// Loop sending data packets until all file data has been sent.
	// Ensure that a final zero size packet is sent if needed.
	// Each data packet sent must wait for an ack from the client.
	// If a data packet gets lost, client retransmits his last ack,
	// and the server will retransmit the last packet sent.
	// Spec: "If a packet gets lost in the
	//   network, the intended recipient will timeout and may retransmit his
	//   last packet (which may be data or an acknowledgment), thus causing
	//   the sender of the lost packet to retransmit that lost packet."

	block_count := (n / dataBlockSize) + 1

	for i := 0;  i <= block_count; i++ {

		// Get the next block.

		start := i * dataBlockSize
		end := (i + 1) * dataBlockSize

		new_block := data_buffer.Bytes()[start:end]

		// Construct a data packet

		var dp tftp.PacketData
		dp.BlockNum = uint16(i + 1)				// TODO downcast is a bad idea...
		dp.Data = new_block

		b := make([]byte, 1024)
		b = dp.Serialize()

		// Set the block number, set BlockAcked to false and start the timeout fn.

		rt := readAddrMap[addr.String()]

		rt.BlockNum = dp.BlockNum
		rt.BlockAcked = false
		go rt.TimeoutTimer()

		// Send the data packet - loop to do retries.

		for {

			pc.WriteTo(b, addr)

			go rt.RetryTimer()

			// Wait for the ack. If we get no ack within 'retry' seconds, resend.
			// We will not retransmit forever - if there is no ack, we must timeout.

			select {
			case <- rt.Acked:
				rt.BlockAcked = true
				break
			case <- rt.Retry:
				continue
			case <- rt.Timeout:
				break
			}
		}

		if !rt.BlockAcked {
			sendError(pc, addr, 0, "Timeout")
			break
		}
	}
}

func deferredMetadataUnlock() {

	fmt.Printf("Releasing Metadata Lock \n")

	lockMetadataChanges.Unlock()
}

func createTrackingEntry(p tftp.PacketRequest) *RequestTracker {

	rt := new(RequestTracker)
	rt.PacketReq = p
	rt.BlockNum = 0
	rt.TransferIncomplete = true
	rt.LastTranferTime = time.Now()
	rt.Acked = make(chan bool, 1)
	rt.Retry = make(chan bool, 1)
	rt.Timeout = make(chan bool, 1)
	rt.BlockAcked = false
	return rt
}




