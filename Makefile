export GO111MODULE=on
BINARY_NAME=synapsevalidator synapsebeacon synapsep2p
SRC=$(shell find . -name "*.go")

all: deps $(BINARY_NAME)
install:
	go install cmd/beacon/synapsebeacon.go
	go install cmd/validator/synapsevalidator.go
	go install cmd/validator/synapsep2p.go
$(BINARY_NAME): $(SRC)
	go build cmd/beacon/synapsebeacon.go
	go build cmd/validator/synapsevalidator.go
	go build cmd/p2p/synapsep2p.go
test: synapsep2p
	go test -v ./...
	go build -o integration_tests test/cmd/integration.go
	./integration_tests
clean:
	go clean
	rm -f $(BINARY_NAME)
deps:
	go build -v ./...
upgrade:
	go get -u