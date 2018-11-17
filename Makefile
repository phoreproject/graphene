export GO111MODULE=on
BINARY_NAME=synapsevalidator synapsebeacon synapsep2p

all: deps build
install:
	go install cmd/beacon/synapsebeacon.go
	go install cmd/validator/synapsevalidator.go
	go install cmd/validator/synapsep2p.go
build:
	go build cmd/beacon/synapsebeacon.go
	go build cmd/validator/synapsevalidator.go
	go build cmd/validator/synapsep2p.go
test:
	go test -v ./...
clean:
	go clean
	rm -f $(BINARY_NAME)
deps:
	go build -v ./...
upgrade:
	go get -u