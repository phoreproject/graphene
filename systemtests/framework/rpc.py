from pb import rpc_pb2
from pb import rpc_pb2_grpc
from pb.google.protobuf import empty_pb2
import grpc

def get_empty() :
    return empty_pb2.Empty()
    
def create_beacon_rpc(rpc_address = '127.0.0.1:11782') :
    channel = grpc.insecure_channel(rpc_address)
    stub = rpc_pb2_grpc.BlockchainRPCStub(channel)
    return stub
    
