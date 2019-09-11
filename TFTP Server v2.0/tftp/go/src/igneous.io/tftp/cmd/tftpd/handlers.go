package main

import (
	"../../../tftp"
	"fmt"
	"net"
)

func handle_read(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle Read Packet Packet: %+v \n", p)
}

func handle_write(pc net.PacketConn, addr net.Addr, p tftp.PacketRequest) {

	fmt.Printf("Handle write Packet Packet: %+v \n", p)
}

func handle_ack(pc net.PacketConn, addr net.Addr, p tftp.PacketAck) {

	// Construct an ack packet and send it to the client

	var pr_send tftp.PacketAck
	pr_send.BlockNum = 1

	b := make([]byte, 1024)
	b = pr_send.Serialize()

	pc.WriteTo(b, addr)
}

func handle_data(pc net.PacketConn, addr net.Addr, p tftp.PacketData) {

	fmt.Printf("Handle Data Packet Packet: %+v \n", p)
}

func handle_error(pc net.PacketConn, addr net.Addr, p tftp.PacketError) {

	fmt.Printf("Handle Error Packet Packet: %+v \n", p)
}

