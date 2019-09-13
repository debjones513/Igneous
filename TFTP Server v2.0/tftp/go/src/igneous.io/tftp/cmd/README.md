Setup
-----
Clone the Igneous repository to
 
 ```/Users/<your user name>/GitHub/Igneous```

Sources are in
 
 ```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go```

Add the sources path to your GOPATH 
(had to remove the second path later, GoLand picked it up as a project config param)

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

