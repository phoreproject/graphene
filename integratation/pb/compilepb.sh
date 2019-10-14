#!/usr/bin/env bash

cd ../../pb
python -m grpc_tools.protoc -I=./ --proto_path=./ --python_out=../systemtests/pb --grpc_python_out=../systemtests/pb rpc.proto
python -m grpc_tools.protoc -I=./ --proto_path=./ --python_out=../systemtests/pb --grpc_python_out=../systemtests/pb p2p.proto
python -m grpc_tools.protoc -I=./ --proto_path=./ --python_out=../systemtests/pb --grpc_python_out=../systemtests/pb google/protobuf/empty.proto
