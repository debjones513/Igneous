package main

import (
	"../tftp"
	"fmt"
	"log"
	"net"
)


func main() {
	//Connect udp
	conn, err := net.Dial("udp", ":69")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Construct a read packet and send it to the server

	var pr_send = new(tftp.PacketRequest)
	pr_send.Mode = "octet"
	pr_send.Op = 1
	pr_send.Filename = "xyz"

	b := make([]byte, 1024)
	b = tftp.Packet.Serialize(pr_send)

	conn.Write(b)

	//simple write - send to server
	//conn.Write([]byte("Hello from client"))

	//simple Read
	//buffer := make([]byte, 1024)
	//conn.Read(buffer)

	// Process a reply fronm the server
	buffer := make([]byte, 1024)
	conn.Read(buffer)

	pr_receive, err := tftp.ParsePacket(buffer)

	fmt.Printf("Parsed Packet Err: %d Packet: %+v \n", err, pr_receive)

	log.Println(buffer)
	//fmt.Printf("Reply from Server: %v \n", buffer)

}