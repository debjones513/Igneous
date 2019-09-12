package main

import (
	"../../../tftp"
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
)

// Tracks the last block sent or received per request
type RequestTracker struct {
	PacketReq tftp.PacketRequest
	BlockNum uint16
	Mux sync.Mutex
}

// Maps file names to file contents - TODO File size is limited, OK since this is just a code exercise
var file_cache map[string]string

// Maps client read addr to the last block transmitted - TODO assumes each client will have a unique port id?
var read_addr_map map[string]RequestTracker

// Maps client write addr to the last block transmitted - TODO assumes each client will have a unique port id?
var write_addr_map map[string]RequestTracker

func init() {
	file_cache = make(map[string]string)
	write_addr_map = make(map[string]RequestTracker)
	read_addr_map = make(map[string]RequestTracker)
}

func handle_read(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Read Packet Packet: %+v \n", p)

	// Create a map entry to find the RequestTracker object given the client address.
	var rt RequestTracker
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

func handle_write(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle write Packet Packet: %+v \n", p)

	// Create a new cache entry for the file.
	// TODO Existing files are overwritten without warning - is this aligned with the requirements?

	file_cache[p.Filename] = ""

	// Create a map entry to find the RequestTracker object given the client address.

	var rt RequestTracker
	rt.PacketReq = p
	rt.BlockNum = 0

	write_addr_map[addr.String()] = rt
}

func handle_data(pc net.PacketConn, addr net.Addr, p tftp.PacketData) {

	// Get the current tracking data.

	var rt RequestTracker
	rt = write_addr_map[addr.String()]

	// Serialize access the code between Mux.Lock() and Mux.Unlock(), per client address.

	rt.Mux.Lock()

	// If we are receiving a data packet, then the client is writing to the server.

	fmt.Printf("Handle Data Packet Packet: %+v \n", p)


	// Send an ack to the client.
	//
	// Once the ack is sent, the client will send the next packet. We lock the meta data, so that if the next packet
	// arrives and begins processing, before process for this packet is complete, the next packet will block at least
	// until we can do the write and update the meta data.
	// TODO Ack tells the client we received the packet, if we fail to write, we panic, and the server fails.

	send_ack(pc, addr, p.BlockNum)

	// Write the next block of data to the in-memory file.

	var new_block bytes.Buffer
	new_block.Write(p.Data)

	var file_data []string
	file_data = append(file_data, file_cache[rt.PacketReq.Filename])	// Current data
	file_data = append(file_data, new_block.String())					// New block

	file_cache[rt.PacketReq.Filename] = strings.Join(file_data, "")

	// Update the meta data.

	rt.BlockNum = p.BlockNum
	write_addr_map[addr.String()] = rt
	rt.Mux.Unlock()
}

func handle_ack(pc net.PacketConn, addr net.Addr, p tftp.PacketAck) {

	fmt.Printf("Handle Ack Packet: %+v \n", p)
}

func handle_error(pc net.PacketConn, addr net.Addr, p tftp.PacketError) {

	fmt.Printf("Handle Error Packet Packet: %+v \n", p)
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



