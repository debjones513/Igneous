package main

import (
	"../../../tftp"
	"bytes"
	"fmt"
	"net"
	"strings"
)

// Tracks the last block sent or received per request
type RequestTracker struct {
	PacketReq tftp.PacketRequest
	BlockNum uint16
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

	// If we are receiving a data packet, then the client is writing to the server.

	fmt.Printf("Handle Data Packet Packet: %+v \n", p)

	// Write the data to the in-memory file.

	var rt RequestTracker
	rt = write_addr_map[addr.String()]

	var new_block bytes.Buffer
	new_block.Write(p.Data)

	var file_data []string
	file_data = append(file_data, file_cache[rt.PacketReq.Filename])	// Current data
	file_data = append(file_data, new_block.String())					// New block

	file_cache[rt.PacketReq.Filename] = strings.Join(file_data, "")

	// Update the meta data

	rt.BlockNum += 1
	write_addr_map[addr.String()] = rt

	// Send an ack to the client.

	

}

func handle_ack(pc net.PacketConn, addr net.Addr, p tftp.PacketAck) {

	fmt.Printf("Handle ack Packet Packet: %+v \n", p)

	// Construct an ack packet and send it to the client

	var pr_send tftp.PacketAck
	pr_send.BlockNum = p.BlockNum

	b := make([]byte, 1024)
	b = pr_send.Serialize()

	pc.WriteTo(b, addr)
}

func handle_error(pc net.PacketConn, addr net.Addr, p tftp.PacketError) {

	fmt.Printf("Handle Error Packet Packet: %+v \n", p)
}


func send_ack(pc net.PacketConn, addr net.Addr, p tftp.PacketAck) {

	fmt.Printf("Send Ack Packet: %+v \n", p)

	// Construct an ack packet and send it to the client

	var pr_send tftp.PacketAck
	pr_send.BlockNum = p.BlockNum

	b := make([]byte, 1024)
	b = pr_send.Serialize()

	pc.WriteTo(b, addr)
}


