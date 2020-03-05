from phore.framework import beaconnode
from phore.framework import shardnode
from phore.framework import tester
from phore.framework import process

import typing
import logging


class ValidatorConfig:
    def __init__(self, beacon_port: int, shard_port: int, validators: str = "0-255"):
        self.beacon_port = beacon_port
        self.shard_port = shard_port
        self.index = -1
        self.networkid = "regtest"
        self.rootkey = "testnet"
        self.validators = validators
        self.validator_executable = tester.get_phore_path("synapsevalidator")
        self.level = "trace"
        self.rpc_port = -1

    def get_args(self) -> typing.List[str]:
        if self.index == -1:
            raise AssertionError("expected index to be set")

        if self.rpc_port == -1:
            raise AssertionError("expected rpc_port to be set")

        args = [
            "-beacon",
            "/ip4/127.0.0.1/tcp/{}".format(self.beacon_port),
            "-shard",
            "/ip4/127.0.0.1/tcp/{}".format(self.shard_port),
            "-networkid",
            self.networkid,
            "-level",
            self.level,
            "-listen",
            "/ip4/127.0.0.1/tcp/{}".format(self.rpc_port),
            "-validators",
            self.validators,
            "-rootkey",
            self.rootkey,
            "-colors"
        ]

        return args

    @staticmethod
    def from_beacon_and_shard(beacon: beaconnode.BeaconNode, shard: shardnode.ShardNode, validators: str = "0-255"):
        return ValidatorConfig(beacon.get_config().rpc_port, shard.get_config().rpc_port, validators)


class ValidatorNode:
    TYPE = "validator"

    def __init__(self, config: ValidatorConfig):
        self._process = None
        self._config = config
        self.started = False

    def start(self):
        logging.info("Starting validator node {}".format(self._config.index))

        self._process = process.Process(
            "validator {}".format(self._config.index),
            self._config.validator_executable,
            *self._config.get_args()
        )
        self._process.start()
        self.started = True
        return True

    def stop(self):
        if self.started:
            self._process.signal_stop()
            self._process.join()

    def get_config(self) -> ValidatorConfig:
        return self._config

    def wait_for_rpc(self):
        pass
        # TODO: uncomment when validator has RPC
        # chan = grpc.insecure_channel("127.0.0.1:{}".format(self._config.rpc_port))
        #
        # ready = grpc.channel_ready_future(chan)
        # while not ready.done():
        #     def test(conn):
        #         pass
        #
        #     chan.subscribe(test, True)
        #
        #     time.sleep(1)
