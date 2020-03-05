import time

import grpc

from phore.framework import process, shardnode
from phore.framework import tester

import typing
import logging

from phore.pb import relayerrpc_pb2_grpc
from google.protobuf import empty_pb2


class RelayerConfig:
    def __init__(self, shard_port: str):
        self.p2p_port = -1
        self.rpc_port = -1
        self.level = "debug"
        self.index = -1
        self.shards = []
        self.relayer_executable = tester.get_phore_path("synapserelayer")
        self.shard_port = shard_port

    def get_args(self) -> typing.List[str]:
        if self.index == -1:
            raise AssertionError("expected index to be set")

        if self.p2p_port == -1:
            raise AssertionError("expected p2p_port to be set")

        if self.rpc_port == -1:
            raise AssertionError("expected p2p_port to be set")

        args = [
            "-shard",
            "/ip4/127.0.0.1/tcp/{}".format(self.shard_port),
            "-rpclisten",
            "/ip4/127.0.0.1/tcp/{}".format(self.rpc_port),
            "-listen",
            "/ip4/127.0.0.1/tcp/{}".format(self.p2p_port),
            "-level",
            self.level,
            "-colors",
            "-shards",
            ",".join(self.shards)
        ]

        return args

    @staticmethod
    def from_shard(shard: shardnode.ShardNode):
        return RelayerConfig(shard.get_config().rpc_port)


def create_config_file(file_name):
    with open(tester.get_phore_path("regtest.json"), "r") as regtest_file:
        regtest_json = regtest_file.read()
    with open(file_name, "w") as file_to_write:
        file_to_write.write(regtest_json)


def snake_to_camel(snake_case: str) -> str:
    components = snake_case.split('_')
    return ''.join([c.title() for c in components])


class RelayerNode:
    TYPE = "relayer"

    def __init__(self, config: RelayerConfig):
        self._config = config
        self._process = None
        self.started = False
        self._rpc = None

    def start(self):
        logging.info("Starting relayer node {}".format(self._config.index))

        self._process = process.Process(
            "relayer {}".format(self._config.index),
            self._config.relayer_executable,
            *self._config.get_args())

        self._process.start()

        self.started = True

    def stop(self):
        if self.started:
            self._process.signal_stop()
            self._process.join()

    def get_config(self) -> RelayerConfig:
        return self._config

    def wait_for_rpc(self):
        chan = grpc.insecure_channel("127.0.0.1:{}".format(self._config.rpc_port))

        ready = grpc.channel_ready_future(chan)
        while not ready.done():
            def test(conn):
                print(conn)
                pass

            chan.subscribe(test, True)

            time.sleep(1)

            self._rpc = relayerrpc_pb2_grpc.RelayerRPCStub(chan)

    def __getattr__(self, item):
        rpc_call = snake_to_camel(item)
        possible_call = getattr(self._rpc, rpc_call, None)
        if possible_call is not None:
            def caller(*args):
                if len(args) == 0:
                    return possible_call(empty_pb2.Empty())
                else:
                    return possible_call(*args)

            return caller
        return None
