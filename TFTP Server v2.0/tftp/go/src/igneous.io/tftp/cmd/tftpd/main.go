package main

import (
	"fmt"
	"log"
"net"
)

/// http://computernetworkingsimplified.in/application-layer/tftp-works/

func main() {
	// listen to incoming udp packets
	pc, err := net.ListenPacket("udp", ":69")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	fmt.Printf("Connection: %+v \n", pc)

	for {
		buf := make([]byte, 1024)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}

		fmt.Printf("Count: %d Addr: %s Err: %d Buffer: %s \n", n, addr, err, buf)

		go serve(pc, addr, buf[:n])
	}

}

func serve(pc net.PacketConn, addr net.Addr, buf []byte) {
	// 0 - 1: ID
	// 2: QR(1): Opcode(4)
	//buf[2] |= 0x80 // Set QR bit

	pc.WriteTo(buf, addr)
}