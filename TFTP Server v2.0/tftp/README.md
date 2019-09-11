In-memory TFTP Server
=====================

This is a simple in-memory TFTP server, implemented in Go.  It is
RFC1350-compliant, but doesn't implement the additions in later RFCs.  In
particular, options are not recognized.

Setup
-----
####Add the path to the project sources to your GOPATH.
#####Example 

Clone the Igneous repository to
 
 ```/Users/<your user name>/GitHub/Igneous```

Sources are in
 
 ```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go```

Add the sources path to your GOPATH

```GOPATH=/usr/local/go/bin:/Users/debjo/GitHub/Igneous/TFTP Server v2.0/tftp/go```

#### Install the packages

CD to the location of the package file sources.

```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go/src/igneous.io/tftp```

Run ```go install```

Verify the tftp package is created

```cd "/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go/pkg/darwin_amd64/igneous.io```

```ls```

You should see

```tftp.a```



Usage
-----
TODO

Testing
-------
TODO

TODO: other relevant documentation
