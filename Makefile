export GO111MODULE=on
BINARY_NAME=synapsevalidator synapsebeacon synapsep2p
SRC=$(shell find . -name "*.go")

# The stupid "go build" can't append the .exe on Windows when using -o, let's do it manually
ifeq ($(OS),Windows_NT)
EXE=.exe
else
EXE=.exe
endif

all: deps $(BINARY_NAME)

install:
	go install cmd/beacon/synapsebeacon.go
	go install cmd/validator/synapsevalidator.go
	go install cmd/validator/synapsep2p.go

$(BINARY_NAME): $(SRC)
	go build cmd/beacon/synapsebeacon.go
	go build cmd/validator/synapsevalidator.go
	go build cmd/p2p/synapsep2p.go

test: unittest integrationtest

unittest:
	go test -v ./...

src_depend: $(SRC)

integrationtests: src_depend
	go build -o integration$(EXE) integrationtests/cmd/integrationtests.go

clean:
	go clean
	rm -f $(BINARY_NAME)

deps:
	go build -v ./...

upgrade:
	go get -u