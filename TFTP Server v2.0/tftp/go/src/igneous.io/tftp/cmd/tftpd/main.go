package main

import (
	"../../../tftp"
	"fmt"
	"log"
	"net"
)

/// http://computernetworkingsimplified.in/application-layer/tftp-works/
// https://tools.ietf.org/html/rfc1350

func main() {

	// Setup to listen on port 69 for all IPs on the local network (localhost only).

	pc, err := net.ListenPacket("udp", ":69")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	fmt.Printf("Connection: %+v \n", pc)
	fmt.Printf("Local Addr: %+v \n", pc.LocalAddr())

	// Handle requests

	for {

		buf := make([]byte, 1024)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}

		fmt.Printf("ReadFrom Count: %d Addr: %s Err: %d\n", n, addr, err)

		p, err := tftp.ParsePacket(buf)

		fmt.Printf("Parsed Packet Err: %d Packet: %+v \n", err, p)

		go serve(pc, addr, buf[:n])
	}

}

func serve(pc net.PacketConn, addr net.Addr, buf []byte) {

	//var b bytes.Buffer
	//b.WriteString("Server Reply")
	//pc.WriteTo(b.Bytes(), addr)


	// Construct an ack packet and send it to the client

	var pr_send tftp.PacketAck
	pr_send.BlockNum = 1

	b := make([]byte, 1024)
	b = pr_send.Serialize()

	pc.WriteTo(b, addr)
}
