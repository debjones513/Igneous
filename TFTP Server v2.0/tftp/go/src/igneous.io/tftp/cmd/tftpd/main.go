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

		serve(pc, addr, buf[:n])
	}

}

func serve(pc net.PacketConn, addr net.Addr, buf []byte) {

	// Parse the packet from the buffer - checks for parsing errors.

	op_code, err := tftp.ParseOpCodeFromPacket(buf)
	if err != nil {
		return
	}

	// Switch on the op code, create the target object type, and forward the packet to the correct handler.

	switch op_code {

	case tftp.OpRRQ:

		var packet_request tftp.PacketRequest
		packet_request.Parse(buf)

		handle_read(pc, addr, packet_request)

	case tftp.OpWRQ:

		var packet_request tftp.PacketRequest
		packet_request.Parse(buf)

		handle_write(pc, addr, packet_request)

	case tftp.OpData:

		var packet_data tftp.PacketData
		packet_data.Parse(buf)

		handle_data(pc, addr, packet_data)

	case tftp.OpAck:

		var packet_ack tftp.PacketAck
		packet_ack.Parse(buf)

		handle_ack(pc, addr, packet_ack)

	case tftp.OpError:

		var packet_error tftp.PacketError
		packet_error.Parse(buf)

		go handle_error(pc, addr, packet_error)

	default:
		err = fmt.Errorf("unexpected packet type %s", op_code)
		return
	}
}
