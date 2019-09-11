package main

import (
	"../tftp"
	"fmt"
	"log"
	"net"
)

func read_file(conn net.Conn, s string) {

	// Construct a read-file packet and send it to the server

	var pr_send = new(tftp.PacketRequest)
	pr_send.Mode = "octet"
	pr_send.Op = 1
	pr_send.Filename = s

	b := make([]byte, 1024)
	b = tftp.Packet.Serialize(pr_send)

	conn.Write(b)

	//simple write - send to server
	//conn.Write([]byte("Hello from client"))
}

func process_read_file(conn net.Conn, s string) {

	//simple Read
	//buffer := make([]byte, 1024)
	//conn.Read(buffer)

	// Process a reply from the server
	buffer := make([]byte, 1024)
	conn.Read(buffer)

	pr_receive, err := tftp.ParsePacket(buffer)

	fmt.Printf("Parsed Packet Err: %d Packet: %+v \n", err, pr_receive)

	//log.Println(buffer)
	//fmt.Printf("Reply from Server: %v \n", buffer)
}

func write_file(conn net.Conn, s string) {

	// Construct a write-file packet and send it to the server

	var pr_send = new(tftp.PacketRequest)
	pr_send.Mode = "octet"
	pr_send.Op = 2
	pr_send.Filename = s

	b := make([]byte, 1024)
	b = tftp.Packet.Serialize(pr_send)

	conn.Write(b)

}

func process_write_file(conn net.Conn, s string) {

	//simple Read
	//buffer := make([]byte, 1024)
	//conn.Read(buffer)

	// Process a reply from the server
	buffer := make([]byte, 1024)
	conn.Read(buffer)

	pr_receive, err := tftp.ParsePacket(buffer)

	fmt.Printf("Parsed Packet Err: %d Packet: %+v \n", err, pr_receive)

	//log.Println(buffer)
	//fmt.Printf("Reply from Server: %v \n", buffer)
}

func main() {

	//Connect to the tftp server on localhost. Listen on port 69.

	conn , err := net.Dial("udp", ":69")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Generate requests

	//write_file(conn, "xyz.txt")
	read_file(conn, "xyz.txt")
	write_file(conn, "xyz.txt")

	log.Println("xyz.txt \n")

}