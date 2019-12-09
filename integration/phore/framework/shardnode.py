import time

import grpc

from phore.framework import process
from phore.framework import tester
from phore.framework import beaconnode

import typing
import logging

from phore.google.protobuf import empty_pb2
from phore.pb import shardrpc_pb2_grpc, shardrpc_pb2


class ShardConfig:
    def __init__(self, beacon_port: str):
        self.beacon_port = beacon_port
        self.level = "trace"
        self.index = -1
        self.rpc_port = -1
        self.p2p_port = -1
        self.network_id = "regtest"
        self.shard_executable = tester.get_phore_path("synapseshard")
        self.initial_shards = []

    def get_args(self) -> typing.List[str]:
        if self.index == -1:
            raise AssertionError("expected index to be set")

        if self.rpc_port == -1:
            raise AssertionError("expected rpc_port to be set")

        if self.p2p_port == -1:
            raise AssertionError("expected p2p_port to be set")

        args = [
            "-beacon",
            "/ip4/127.0.0.1/tcp/{}".format(self.beacon_port),
            "-rpclisten",
            "/ip4/127.0.0.1/tcp/{}".format(self.rpc_port),
            "-listen",
            "/ip4/127.0.0.1/tcp/{}".format(self.p2p_port),
            "-networkid",
            self.network_id,
            "-level",
            self.level,
            "-colors",
        ]

        if len(self.initial_shards) > 0:
            args += [
                "-track",
                ",".join(self.initial_shards)
            ]

        return args

    @staticmethod
    def from_beacon(beacon: beaconnode.BeaconNode):
        return ShardConfig(beacon.get_config().rpc_port)


def snake_to_camel(snake_case: str) -> str:
    components = snake_case.split('_')
    return ''.join([c.title() for c in components])


class ShardNode:
    TYPE = "shard"

    def __init__(self, config: ShardConfig):
        self._config = config
        self._process = None
        self.started = False

    def start(self):
        logging.info("Starting shard node {}".format(self._config.index))

        self._process = process.Process(
            "shard {}".format(self._config.index),
            self._config.shard_executable,
            *self._config.get_args()
        )
        self._process.start()

        self.started = True

    def stop(self):
        if self.started:
            self._process.signal_stop()
            self._process.join()

    def get_config(self) -> ShardConfig:
        return self._config

    def wait_for_rpc(self):
        chan = grpc.insecure_channel("127.0.0.1:{}".format(self._config.rpc_port))

        ready = grpc.channel_ready_future(chan)
        while not ready.done():
            def test(conn):
                pass

            chan.subscribe(test, True)

            time.sleep(1)

            self._rpc = shardrpc_pb2_grpc.ShardRPCStub(chan)

    def wait_for_slot(self, slot_to_wait: int, shard_id: int):
        while True:
            slot_number_response = self._rpc.GetSlotNumber(shardrpc_pb2.SlotNumberRequest(ShardID=shard_id))
            if slot_number_response.TipSlot >= slot_to_wait:
                break
            time.sleep(1)

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
