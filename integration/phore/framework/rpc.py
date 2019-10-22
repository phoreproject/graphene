import beaconrpc_pb2
import beaconrpc_pb2_grpc
from google.protobuf import empty_pb2
import grpc

def get_empty():
    return empty_pb2.Empty()

def create_beacon_rpc(rpc_address = '127.0.0.1:11782'):
    channel = grpc.insecure_channel(rpc_address)
    stub = beaconrpc_pb2_grpc.BlockchainRPCStub(channel)
    return stub

