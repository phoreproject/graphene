export GO111MODULE=on
BINARY_NAME=grapheneshard graphenevalidator graphenebeacon graphenekey grapheneexplorer graphene graphenerelayer
SRC=$(shell find . -name "*.go")
TESTS=$(shell find integration/phore/tests -name "*.py" | sed "s/.*\//integration_test_/" | sed "s/\.py//")

# The stupid "go build" can't append the .exe on Windows when using -o, let's do it manually
ifeq ($(OS),Windows_NT)
EXE=.exe
else
EXE=
endif

all: deps $(BINARY_NAME)

install:
	go install cmd/beacon/graphenebeacon.go
	go install cmd/validator/graphenevalidator.go
	go install cmd/keygen/graphenekey.go
	go install cmd/graphene/graphene.go
	go install cmd/relayer/graphenerelayer.go

grapheneshard: $(SRC)
	go build cmd/shard/grapheneshard.go

graphenebeacon: $(SRC)
	go build cmd/beacon/graphenebeacon.go

graphenevalidator: $(SRC)
	go build cmd/validator/graphenevalidator.go

graphenekey: $(SRC)
	go build cmd/keygen/graphenekey.go

grapheneexplorer: $(SRC)
	go build explorer/cmd/grapheneexplorer.go

graphene: $(SRC)
	go build cmd/graphene/graphene.go

graphenerelayer: $(SRC)
	go build cmd/relayer/graphenerelayer.go

test: unittest integrationtests

unittest:
	go test -v ./...

src_depend: $(SRC)

integrationtests: $(TESTS)

integration_test_%: integration/phore/tests/%.py $(BINARY_NAME) 
	PYTHONPATH=integration python3 $<

clean:
	go clean
	rm -f $(BINARY_NAME)

deps:
	go build -v ./...

upgrade:
	go get -u
