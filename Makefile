export GO111MODULE=on
BINARY_NAME=synapsevalidator synapsebeacon synapsekey
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

$(BINARY_NAME): $(SRC)
	go build cmd/beacon/synapsebeacon.go
	go build cmd/validator/synapsevalidator.go
	go build cmd/keygen/synapsekey.go

test: unittest integrationtests

unittest:
	go test -v ./...

src_depend: $(SRC)

integrationtests: src_depend
	go build -o integration$(EXE) integrationtests/cmd/integrationtests.go
	./integration$(EXE)

clean:
	go clean
	rm -f $(BINARY_NAME)

deps:
	go build -v ./...

upgrade:
	go get -u
