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

const DataBlockSize = 512

// Tracks the last block sent or received per request, whether or not the request is incomplete, and the timestamp
// for the last request was processed. The last two fields are used to cleanup stale entries.

type RequestTracker struct {
	PacketReq tftp.PacketRequest
	BlockNum uint16
	Acked bool
	Mux sync.Mutex
	TransferIncomplete bool
	LastTranferTime time.Time
}

// Maps file names to file contents - TODO File size is limited, OK since this is just a code exercise

var file_cache map[string]string

// Maps client addr to the last block transmitted.

var read_addr_map map[string]*RequestTracker

// Maps client addr to the last block transmitted.

var write_addr_map map[string]*RequestTracker

// Globals to enum lists

var files *map[string]string
var read_addrs *map[string]*RequestTracker
var write_addrs *map[string]*RequestTracker


func init() {
	file_cache = make(map[string]string)
	write_addr_map = make(map[string]*RequestTracker)
	read_addr_map = make(map[string]*RequestTracker)

	files = &file_cache
	read_addrs = &read_addr_map
	write_addrs = &write_addr_map
}

func handle_read(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Read Packet: %+v \n", p)

	// Lookup the file in our cache, return an error if the file is not found.

	if _, ok := file_cache[p.Filename]; ok == false {
		send_error(pc, addr, 1, "File not found.")
	}

	// Create a map entry to find the RequestTracker object given the client address when sending data packets.

	read_addr_map[addr.String()] = create_tracking_entry(p)

	// Spec: "RRQ ... packets are acknowledged by DATA or ERROR packets. No ack needed here,
	// just send the first data packet."

	send_data(pc, addr, p)
}

func handle_write(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Write Packet: %+v \n", p)

	// Lookup the file in our cache, return an error if the file already exists.

	if _, ok := file_cache[p.Filename]; ok == true {
		send_error(pc, addr, 1, "File already exists.")
		return
	}

	// Create a new cache entry for the file.

	file_cache[p.Filename] = ""

	// Create a map entry. Used to find the RequestTracker object given the client address during data packet transfers.

	write_addr_map[addr.String()] = create_tracking_entry(p)

	// Spec: "A WRQ is acknowledged with an ACK packet with block number set to zero."

	send_ack(pc, addr, 0)

	fmt.Printf("Handle Write Packet: %+v \n  %+v \n  %+v \n", files, read_addrs, write_addrs)
}

func handle_data(pc net.PacketConn, addr net.Addr, p tftp.PacketData) {

	// If we are receiving a data packet, then the client is writing to the server.

	fmt.Printf("Handle Data Packet: %+v \n", p)

	// Lookup the RequestTracker object, if it is not found, return an error.
	//
	// Spec: "TFTP recognizes only one error condition that does not cause
	//   termination, the source port of a received packet being incorrect.
	//   In this case, an error packet is sent to the originating host."

	if _, ok := write_addr_map[addr.String()]; ok == false {
		send_error(pc, addr, 5, "Unknown transfer ID.")
		return
	}

	// Get the current tracking data.

	rt := write_addr_map[addr.String()]

	// Serialize access to the code between Mux.Lock() and Mux.Unlock(), per client address.

	rt.Mux.Lock()
	defer deferred_unlock(rt)

	fmt.Printf("Data Packet Lock Taken: %+v Client: %s Tracker: %+v \n", p, addr.String(), rt)

	// We ack'ed the last data packet before processing was completed, to enable better perf.
	// Check for duplicate blocks being sent, and that the last packet written corresponds to the data block
	// preceding the current data block.

	if rt.BlockNum == p.BlockNum {
		// Duplicate block resent - maybe we did not ack quick enough. Ignore it, we already wrote this block.
		// TODO Should we go ahead and ack this block a second time?
		send_ack(pc, addr, p.BlockNum)
		return
	} else if rt.BlockNum + 1 != p.BlockNum {
		send_error(pc, addr, 0, "Missing data block in transfer sequence.")
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

	send_ack(pc, addr, p.BlockNum)

	// If this is the final transfer packet, and it is empty, delete the Tracker entry and return.

	if len(p.Data) == 0 {
		delete(write_addr_map, addr.String())
		return
	}

	// Write the next block of data to the in-memory file.

	var new_block bytes.Buffer
	new_block.Write(p.Data)

	// TODO memory now holds two copies of the file data ...
	// TODO instead, use the block number to multiply by DataBlockSize and find the correct position, maybe slice functions...

	var file_data []string
	file_data = append(file_data, file_cache[rt.PacketReq.Filename])	// Current data
	file_data = append(file_data, new_block.String())					// New block

	file_cache[rt.PacketReq.Filename] = strings.Join(file_data, "")

	// Update the meta data with the last block written and timestamp.

	rt.BlockNum = p.BlockNum
	rt.LastTranferTime = time.Now()

	// If this is the final transfer packet, delete the RequestTracker entry

	if len(p.Data) < DataBlockSize {
		delete(write_addr_map, addr.String())
	}

	// TODO If the transfer for some reason stops before we receive a final transfer packet, then the file is
	// TODO partially written. Added a bit to the RequestTracker to signify incomplete transfer. At some point
	// TODO these should be cleaned up... and this case should not block a second transfer of the same file.
	// TODO See item #7 in the spec...

	fmt.Printf("Handle Write Packet: %+v \n  %+v \n  %+v \n", files, read_addrs, write_addrs)
}

func handle_ack(pc net.PacketConn, addr net.Addr, p tftp.PacketAck) {

	fmt.Printf("Handle Ack Packet: %+v \n", p)

	// If we received an ack, it is in response to a data packet being sent.
	// Update the RequestTracking data.
	// TODO use a channel to signal the send_data fn to send the next block.

	if _, ok := read_addr_map[addr.String()]; ok == false {
		send_error(pc, addr, 5, "Unknown transfer ID.")
		return
	}

	read_addr_map[addr.String()].Acked = true

}

func handle_error(pc net.PacketConn, addr net.Addr, p tftp.PacketError) {

	fmt.Printf("Handle Error Packet Packet: %+v \n", p)

	// TODO See item #7 in the spec...
}

func deferred_unlock(rt *RequestTracker) {

	fmt.Printf("Releasing RequestTracker Lock %+v \n", rt)

	rt.Mux.Unlock()
}

func send_ack(pc net.PacketConn, addr net.Addr, block_num uint16) {

	fmt.Printf("Send Ack Packet: %+v \n", block_num)

	// Construct an ack packet and send it to the client

	var ack_packet tftp.PacketAck
	ack_packet.BlockNum = block_num

	b := make([]byte, 1024)
	b = ack_packet.Serialize()

	pc.WriteTo(b, addr)
}

func send_error(pc net.PacketConn, addr net.Addr, Code uint16, Msg  string) {

	fmt.Printf("Handle Error Packet: %+v \n", addr)

	// Construct an error packet and send it to the client

	var error_packet tftp.PacketError
	error_packet.Code = Code
	error_packet.Msg = Msg

	b := make([]byte, 1024)
	b = error_packet.Serialize()

	pc.WriteTo(b, addr)
}

func send_data(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Read Packet Packet: %+v \n", p)

	// Lookup the file in our cache.

	data := file_cache[p.Filename]

	// The return value n is the length of the buffer; err is always nil. If the
	// buffer becomes too large, Write will panic with ErrTooLarge.

	var data_buffer bytes.Buffer
	n, _ := data_buffer.WriteString(data)

	// Loop sending data packets until all file data has been sent.
	// Ensure that a final zero size packet is sent.
	// Each data packet sent must wait of an ack from the client.
	// If a data packet gets lost, client retransmits his last ack,
	// and the server will retransmit the last packet sent.

	block_count := (n / DataBlockSize) + 1

	for i := 0;  i <= block_count; i++ {

		// Get the next block.

		start := i * DataBlockSize
		end := (i + 1) * DataBlockSize

		new_block := data_buffer.Bytes()[start:end]

		// Construct a data packet and send it to the client

		var dp tftp.PacketData
		dp.BlockNum = uint16(i + 1)				// TODO downcast is a bad idea...
		dp.Data = new_block

		b := make([]byte, 1024)
		b = dp.Serialize()

		// Wait for the ack. If we get no ack within 'timeout' seconds, resend.
		// We will not retransmit forever - if there is no ack, we must timeout.

		// while no_ack {
		rt := read_addr_map[addr.String()]
		rt.BlockNum = dp.BlockNum
		rt.Acked = false

		pc.WriteTo(b, addr)

		// }

	}
}

func create_tracking_entry(p tftp.PacketRequest) *RequestTracker {

	rt := new(RequestTracker)
	rt.PacketReq = p
	rt.BlockNum = 0
	rt.TransferIncomplete = true
	rt.LastTranferTime = time.Now()

	return rt
}




