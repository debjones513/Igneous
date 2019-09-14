In-memory TFTP Server
=====================

This is a simple in-memory TFTP server, implemented in Go.  It is
RFC1350-compliant, but doesn't implement the additions in later RFCs.  In
particular, options are not recognized.

See https://tools.ietf.org/html/rfc1350

Caveat
-----
!!! THIS IS NOT PRODUCTION CODE!!!

This code is written as a coding exercise. No design review, limited testing, and comments targeting 
an exercise - callouts where further work would need to be done, design points that would need to be further 
considered, etc.
 
Minimal testing was done using only a single server.

Coding style uses liberal white space. I know many have strong opinions about code style - I have no problem 
conforming to the style that is preferred by the team.

The comments are a bit wordy in places. Since this is a code exercise (not production code), the intention is to 
give the reviewers a little insight into what I was thinking.

TODO's call out some of the work that would need to be done to finalize a production ready service.

The code uses fmt to log to stdout for debugging purposes - this is not production code.

Usage
-----
Logs are located in the service binary directory. There is a request log, and a debug log.

If you are running this code under a debugger, you will want to set the TFTP client timeouts to a value greater 
than the defaults. See ```rexmt``` and ```timeouts``` values for Mac.

The listening port is set to 69 in the server sources. If you use port 69, you will have to shutdown any local 
TFTP service before running the code exercise service. I tested using port 9969, which requires a code change 
in main.go and a rebuild.

####On Mac
The tftp client app is pre-installed on your Mac.

Run ```tftp```

If using port 9969, set the host and port at startup. Replace the command above with

```tftp localhost 9969```

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
Testing was done first using a quick client app that will send a packet. Once a simple packet transfer was 
verified, and the server was stubbed out a little farther, switched over to testing using the TFTP client 
that ships with Mac. The code in the 'client' folder can be ignored.

Tested using port 9969 rather than stopping the TFTP service that ships with Mac.

No GoLang testing done yet. I have worked on teams that used GoLang, but we did not use the test functionality. 
I will have to do some reading there.

Tested using various files, and the ```diff``` tool. For example upload a file on disk to my server, 
rename local file, download file from my server and diff.

The testing I have done is pretty minimal. 

#### Mac TFTP Client idiosyncracies
If I call ```get xyz```, and that file exists in my local directory, but does not exist on 
my TFTP server, my server returns an error packet (which is ack'ed) and the Mac client zeros out the local file.

Not sure if this is expected behavior...








