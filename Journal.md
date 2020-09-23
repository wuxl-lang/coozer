## protocol buffers
https://developers.google.com/protocol-buffers/docs/gotutorial

    $ brew install protobuf
    $ go get google.golang.org/protobuf/cmd/protoc-gen-go 
    $ go install google.golang.org/protobuf/cmd/protoc-gen-go

Config $GOPATH/bin to $PATH

    $ protoc --go_out=./ ./msg.proto