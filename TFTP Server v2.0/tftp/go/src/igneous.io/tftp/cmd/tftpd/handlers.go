package main

import (
	"../../../tftp"
	"bytes"
	"fmt"
	"net"
)

type RequestTracker struct {
	PacketReq tftp.PacketRequest
	BlockNum uint16
}

// Maps file names to file data
var file_cache map[string]string

// Maps client read addr to the last block transmitted
var write_addr_map map[string]RequestTracker

// Maps client write addr to the last block transmitted
var read_addr_map map[string]RequestTracker

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

	file_cache[p.Filename] = ""

	// Create a map entry to find the RequestTracker object given the client address.

	var rt RequestTracker
	rt.PacketReq = p
	rt.BlockNum = 0

	write_addr_map[addr.String()] = rt

	// Now loop sending data packets, and waiting for an ack for each packet.

	//for {
	//	file_cache[p.Filename]
	//}
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

func handle_data(pc net.PacketConn, addr net.Addr, p tftp.PacketData) {

	fmt.Printf("Handle Data Packet Packet: %+v \n", p)

	// Now loop sending data packets, and waiting for an ack for each packet.

	//for {
	//	file_cache[p.Filename]
	//}

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


