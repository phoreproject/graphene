#!/usr/bin/env bash

mkdir -p ./pb
python -m grpc_tools.protoc -I=./ --proto_path=../../pb --python_out=./pb --grpc_python_out=./pb ../../pb/*.proto
