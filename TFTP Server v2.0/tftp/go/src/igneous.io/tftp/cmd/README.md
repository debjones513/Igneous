Setup
-----
Clone the Igneous repository to a directory of your choice, the example uses
 
 ```/Users/<your user name>/GitHub/Igneous```

Sources are in
 
 ```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go```

Add the sources path to your GOPATH 
(had to remove the second path later, GoLand picked it up as a project config param).

```GOPATH=/usr/local/go/bin:/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go```

I have the latest version of GoLang installed.


### Install the packages

I added one function to the tftp package. The latest version is of the package checked in, so you should 
not need to rebuild. To rebuild:

CD to the location of the package file sources.

```cd "/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go/src/igneous.io/tftp"```

Run ```go install```

Verify the tftp package is created

```cd "/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go/pkg/darwin_amd64/igneous.io```

```ls```

You should see

```tftp.a```


### GoLand project config params

#### Server
Choose the ```go build``` configuration template and create a new instance.


Run Kind : ```Directory```

Directory: ```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go/src/igneous.io/tftp/cmd/tftpd```

Output Directory:```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go/bin```

Run After Build: checked

Working Directory: ```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/```

...

Module: ```tftp```


#### Client
Choose the ```go build``` configuration template and create a new instance.


Run Kind : ```File```

Files: ```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go/src/igneous.io/client/client.go```

Run After Build: checked

Working Directory: ```/Users/<your user name>/GitHub/Igneous/TFTP Server v2.0/tftp/go/src/igneous.io/client/```

...

Module: ```client```



