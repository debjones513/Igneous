package main

import (
	"../../../tftp"
	"log"
	"net"
	"os"
)

/// http://computernetworkingsimplified.in/application-layer/tftp-works/
// https://tools.ietf.org/html/rfc1350

var requestLog *log.Logger
var debugLog  *log.Logger

func main() {

	// Setup logs.

	fileRequest, fileDebug :=  setupLogFiles()
	defer fileRequest.Close()
	defer fileDebug.Close()

	requestLog = log.New(fileRequest, "", log.Ldate | log.Ltime)
	debugLog = log.New(fileDebug, "", log.Ldate | log.Ltime)

	// Listen on port 69 for all IPs on the local network (localhost only).

	pc, err := net.ListenPacket("udp", ":69")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	debugLog.Printf("Connection: %+v \n", pc)
	debugLog.Printf("Local Addr: %+v \n", pc.LocalAddr())

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

	// Parse the op code from the buffer.

	op_code, err := tftp.ParseOpCodeFromPacket(buf)
	if err != nil {
		return
	}

	// Switch on the op code, create the target object type, and forward the packet to the correct handler.

	switch op_code {

	case tftp.OpRRQ:

		requestLog.Println("Read")

		var packetRequest tftp.PacketRequest
		packetRequest.Parse(buf)

		go handleRead(pc, addr, packetRequest)

	case tftp.OpWRQ:

		requestLog.Println("Write")

		var packetRequest tftp.PacketRequest
		packetRequest.Parse(buf)

		go handleWrite(pc, addr, packetRequest)

	case tftp.OpData:

		var packetData tftp.PacketData
		packetData.Parse(buf)

		go handleData(pc, addr, packetData)

	case tftp.OpAck:

		var packetAck tftp.PacketAck
		packetAck.Parse(buf)

		go handleAck(pc, addr, packetAck)

	case tftp.OpError:

		// TFTP recognizes only one error condition that does not cause
		//   termination, the source port of a received packet being incorrect.
		//   In this case, an error packet is sent to the originating host.

		var packetError tftp.PacketError
		packetError.Parse(buf)

		go handleError(pc, addr, packetError)

	default:

		requestLog.Printf("Unexpected packet type %s", op_code)
		return
	}
}

func setupLogFiles() (*os.File, *os.File) {

	// Setup logs.

	fileRequest, errRequest := os.Create("tftp_request.log")
	if errRequest != nil {
		log.Fatal(errRequest)
	}

	fileDebug, errDebug := os.Create("tftp_debug.log")
	if errDebug != nil {
		log.Fatal(errDebug)
	}

	return fileRequest, fileDebug
}

