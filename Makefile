export GO111MODULE=on
BINARY_NAME=synapsevalidator synapsebeacon synapsekey synapseexplorer synapse synapserelayer
SRC=$(shell find . -name "*.go")

# The stupid "go build" can't append the .exe on Windows when using -o, let's do it manually
ifeq ($(OS),Windows_NT)
EXE=.exe
else
EXE=
endif

all: deps $(BINARY_NAME)

install:
	go install cmd/beacon/synapsebeacon.go
	go install cmd/validator/synapsevalidator.go
	go install cmd/keygen/synapsekey.go
	go install cmd/synapse/synapse.go
	go install cmd/relayer/synapserelayer.go

synapsebeacon: $(SRC)
	go build cmd/beacon/synapsebeacon.go

synapsevalidator: $(SRC)
	go build cmd/validator/synapsevalidator.go

synapsekey: $(SRC)
	go build cmd/keygen/synapsekey.go

synapseexplorer: $(SRC)
	go build explorer/cmd/synapseexplorer.go

synapse: $(SRC)
	go build cmd/synapse/synapse.go

synapserelayer: $(SRC)
	go build cmd/relayer/synapserelayer.go

test: unittest integrationtests

unittest:
	go test -v ./...

src_depend: $(SRC)

integrationtests: src_depend

clean:
	go clean
	rm -f $(BINARY_NAME)

deps:
	go build -v ./...

upgrade:
	go get -u
