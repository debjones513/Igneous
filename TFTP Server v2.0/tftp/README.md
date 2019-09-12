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
1. The net.Addr struct will have a different port for each client - true?
2. Uploading the same file twice will overwrite the existing file, file data is updated.
3. Ack only implies that we received the packet, not that we successfully wrote the packet - true?

