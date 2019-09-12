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

// Tracks the last block sent or received per request
type RequestTracker struct {
	PacketReq tftp.PacketRequest
	BlockNum uint16
	Mux sync.Mutex
	TransferIncomplete bool
	LastTranferTime time.Time
}

// Maps file names to file contents - TODO File size is limited, OK since this is just a code exercise
var file_cache map[string]string

// Maps client read addr to the last block transmitted - TODO assumes each client will have a unique port id?
var read_addr_map map[string]*RequestTracker

// Maps client write addr to the last block transmitted - TODO assumes each client will have a unique port id?
var write_addr_map map[string]*RequestTracker

func init() {
	file_cache = make(map[string]string)
	write_addr_map = make(map[string]*RequestTracker)
	read_addr_map = make(map[string]*RequestTracker)
}

func handle_read(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Read Packet: %+v \n", p)

	// Lookup the file in our cache.

	if _, ok := file_cache[p.Filename]; ok == false {
		send_error(pc, addr, 1, "File not found.")
	}

	// Create a map entry to find the RequestTracker object given the client address when sending data packets.

	rt := new(RequestTracker)
	rt.PacketReq = p
	rt.BlockNum = 0

	read_addr_map[addr.String()] = rt

	// Loop sending data blocks, with a final zero length block to terminate the transfer.
	// After each data send, we wait for an ack from the client before continuing.

	for {

		var data_buffer bytes.Buffer

		// loop sending data packets until all file data has been sent.
		// Ensure that a final zero size packet is sent.
		// Each data packet sent must wait of an ack from the client.
		// If a data packet gets lost, client retransmits his last ack,
		// and the server will retransmit the last packet sent.

		data_buffer.WriteString("This is file xyz's text")

		// Construct a data packet and send it to the client

		var pr_send tftp.PacketData
		pr_send.BlockNum = 1
		pr_send.Data = data_buffer.Bytes()

		b := make([]byte, 1024)
		b = pr_send.Serialize()

		//data_buffer.Reset()
		pc.WriteTo(b, addr)
	}
}

func handle_write(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Write Packet: %+v \n", p)

	// Lookup the file in our cache.

	if _, ok := file_cache[p.Filename]; ok == true {
		send_error(pc, addr, 1, "File already exists.")
	}

	// Create a new cache entry for the file.

	file_cache[p.Filename] = ""

	// Create a map entry to find the RequestTracker object given the client address during data packet transfers.

	rt := new(RequestTracker)
	rt.PacketReq = p
	rt.BlockNum = 0
	rt.TransferIncomplete = true
	rt.LastTranferTime = time.Now()

	write_addr_map[addr.String()] = rt

	// Spec: A WRQ is acknowledged with an ACK packet having a block number of zero.
}

func handle_data(pc net.PacketConn, addr net.Addr, p tftp.PacketData) {

	// If we are receiving a data packet, then the client is writing to the server.

	fmt.Printf("Handle Data Packet: %+v \n", p)

	// Get the current tracking data.

	rt := write_addr_map[addr.String()]

	// Serialize access the code between Mux.Lock() and Mux.Unlock(), per client address.

	rt.Mux.Lock()
	defer deferred_unlock(rt)

	fmt.Printf("Data Packet Lock Taken Packet: %+v Client: %s Tracker: %+v \n", p, addr.String(), rt)

	// We acked the last data packet before being sure it was properly written, to enable better perf.
	// Check that the last packet written corresponds to the data block preceeding the current data block.
	// TODO If it does not, then we should terminate the connection - by sending an error packet?

	if rt.BlockNum + 1 != p.BlockNum {
		send_error(pc, addr, 0, "Missing data block in transfer sequence.")
		return
	}

	// Send an ack to the client.
	//
	// Once the ack is sent, the client will send the next packet. We lock the meta data, so that if the next packet
	// arrives and begins processing, before processing for this packet is complete, the next packet will block
	// we can do the write and update the meta data.  This way we try to achieve the state where we constantly have
	// the next packet available for processing. Note that only 1 packet, the next packet, is available.
	// This method has the potential to eliminate all time that would be spent waiting for the next packet to be
	// transmitted over the wire - for a large number if packets, this could be a significant perf benefit.
	// On the other hand, if packets are often received but malformed resulting in an error applying the data,
	// then we would put the weight on successful completion rather than speed, and wait to ack.
	// TODO Ack tells the client we received the packet, if we fail to write, we panic, and the server fails, or,
	// TODO we send an error packet to terminate the transfer - correct?

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

	var file_data []string
	file_data = append(file_data, file_cache[rt.PacketReq.Filename])	// Current data
	file_data = append(file_data, new_block.String())					// New block

	file_cache[rt.PacketReq.Filename] = strings.Join(file_data, "")

	// Update the meta data with the last block written and timestamp.

	rt.BlockNum = p.BlockNum
	rt.LastTranferTime = time.Now()

	// If this is the final transfer packet, delete the RequestTracker entry

	if len(p.Data) < 512 {
		delete(write_addr_map, addr.String())
	}

	// TODO If the transfer for some reason stops before we receive a final transfer packet, then the file is
	// TODO partially written. Add a bit to the RequestTracker to signify incomplete transfer. At some point
	// TODO these should be cleaned up... and this case should not block a second transfer of the same file.
}

func handle_ack(pc net.PacketConn, addr net.Addr, p tftp.PacketAck) {

	fmt.Printf("Handle Ack Packet: %+v \n", p)
}

func handle_error(pc net.PacketConn, addr net.Addr, p tftp.PacketError) {

	fmt.Printf("Handle Error Packet Packet: %+v \n", p)
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

	// Construct an ack packet and send it to the client

	var error_packet tftp.PacketError
	error_packet.Code = Code
	error_packet.Msg = Msg

	b := make([]byte, 1024)
	b = error_packet.Serialize()

	pc.WriteTo(b, addr)
}

func send_data(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Read Packet Packet: %+v \n", p)

	// Create a map entry to find the RequestTracker object given the client address.
	rt := new(RequestTracker)
	rt.PacketReq = p
	rt.BlockNum = 0

	read_addr_map[addr.String()] = rt

	// Lookup the file in our cache

	//for {

	var data_buffer bytes.Buffer

	// loop sending data packets until all file data has been sent.
	// Ensure that a final zero size packet is sent.
	// Each data packet sent must wait of an ack from the client.
	// If a data packet gets lost, client retransmits his last ack,
	// and the server will retransmit the last packet sent.

	data_buffer.WriteString("This is file xyz's text")

	// Construct a data packet and send it to the client

	var pr_send tftp.PacketData
	pr_send.BlockNum = 1
	pr_send.Data = data_buffer.Bytes()

	b := make([]byte, 1024)
	b = pr_send.Serialize()

	//data_buffer.Reset()
	pc.WriteTo(b, addr)
	//}
}





