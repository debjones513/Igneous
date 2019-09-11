package main

import (
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

	//simple write
	conn.Write([]byte("Hello from client"))

	//simple Read
	buffer := make([]byte, 1024)
	conn.Read(buffer)
	log.Println(buffer)

}