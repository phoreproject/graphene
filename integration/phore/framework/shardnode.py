import time

import grpc

from phore.framework import process
from phore.framework import tester
from phore.framework import beaconnode

import typing
import sys
import logging


class ShardConfig:
    def __init__(self, beacon_port: str):
        self.beacon_port = beacon_port
        self.level = "trace"
        self.index = -1
        self.rpc_port = -1
        self.beacon_executable = tester.get_phore_path("synapseshard")

    def get_args(self) -> typing.List[str]:
        if self.index == -1:
            raise AssertionError("expected index to be set")

        if self.rpc_port == -1:
            raise AssertionError("expected rpc_port to be set")

        args = [
            "-beacon",
            "/ip4/127.0.0.1/tcp/{}".format(self.beacon_port),
            "-listen",
            "/ip4/127.0.0.1/tcp/{}".format(self.rpc_port),
            "-level",
            self.level,
            "-colors",
        ]

        return args

    @staticmethod
    def from_beacon(beacon: beaconnode.BeaconNode):
        return ShardConfig(beacon.get_config().rpc_port)

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
            self._config.beacon_executable,
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
