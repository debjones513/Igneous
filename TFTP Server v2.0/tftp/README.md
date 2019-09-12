In-memory TFTP Server
=====================

This is a simple in-memory TFTP server, implemented in Go.  It is
RFC1350-compliant, but doesn't implement the additions in later RFCs.  In
particular, options are not recognized.

See https://tools.ietf.org/html/rfc1350

Usage
-----
####On Mac
The tftp client app is pre-installed on your Mac.

Run ```tftp```

Set the Mode to binary ('binary' and 'octet' are interchangable terms).

```tftp> binary```

Upload a file to the tftp server's in-memory cache.

```tftp> put xyz.txt```

Read a file from the tftp server's in-memory cache.

```tftp> get xyz.txt```

####On Linux

TODO

Testing
-------
TODO

TODO: other relevant documentation

Questions
-----
1. The net.Addr struct will have a different port for each client on localhost - true? Yes
Spec: "In order to create a connection, each end of the connection chooses a
          TID for itself, to be used for the duration of that connection.  The
          TID's chosen for a connection should be randomly chosen, so that the
          probability that the same number is chosen twice in immediate
          succession is very low.  Every packet has associated with it the two
          TID's of the ends of the connection, the source TID and the
          destination TID.  These TID's are handed to the supporting UDP (or
          other datagram protocol) as the source and destination ports. "

2. Uploading the same file twice will overwrite the existing file, file data is updated. No - this is an error case ...

```Error 6         File already exists.```

3. Ack only implies that we received the packet, not that we successfully wrote the packet - true? Or is it better\OK 
to wait until the write is done to ack - less concurrency, but if the packet data were corrupted (nil?), failing to ack 
will trigger a resend. 

If we want the concurrency, then we ack right away, but then we cannot get a packet resend from the client to retry.

Spec: "
Most errors cause termination of the connection.  An error is
   signalled by sending an error packet.  This packet is not
   acknowledged, and not retransmitted (i.e., a TFTP server or user may
   terminate after sending an error message), so the other end of the
   connection may not get it.  Therefore timeouts are used to detect
   such a termination when the error packet has been lost. 
    
Errors are caused by three types of events: not being able to satisfy the
   request (e.g., file not found, access violation, or no such user),
   receiving a packet which cannot be explained by a delay or
   duplication in the network (e.g., an incorrectly formed packet), and
   losing access to a necessary resource (e.g., disk full or access
   denied during a transfer).
   
TFTP recognizes only one error condition that does not cause **termination** (terminatiuon of the connection I assume?), 
the source port of a received packet 
being incorrect. In this case, an error packet is sent to the originating host."

Spec "lock step acknowledgement provides flow control and eliminates the need to reorder 
incoming data packets.". So the below is moot...

Technically, there really is no problem with writing packet #2 before packet #1, but we need to know
when we are done... and we need to know that all packets were written, if we declare success...
Could just queue packets for processing and ack as they are received and queued, sort the queue,
but would have to track when we are 'done'.

4.
Dest. Port      Picked by destination machine (69 for RRQ or WRQ).
So we could handle data and acks on some other port...

5.  SPEC: "A data packet of less than 512 bytes signals termination of a transfer."
An error packet also signals termination of transfer.

```Error 5         Unknown transfer ID.```






