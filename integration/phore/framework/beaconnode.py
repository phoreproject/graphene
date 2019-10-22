import time

import grpc

from phore.framework import process
from phore.framework import tester

import os
import pathlib
import typing
import sys
import logging

from phore.pb import beaconrpc_pb2_grpc, beaconrpc_pb2
from google.protobuf import empty_pb2


class BeaconConfig:
    def __init__(self, index: int, data_directory: str, p2p_port: int, rpc_port: int):
        self.datadir = data_directory
        self.chaincfg = "testnet"
        self.p2p_port = p2p_port
        self.rpc_port = rpc_port
        self.connect = []
        self.level = "debug"
        self.index = index
        self.beacon_executable = tester.get_phore_path("synapsebeacon")
        self.genesis_time = "+5"

    def get_args(self) -> typing.List[str]:
        args = [
            "-datadir",
            self.datadir,
            "-rpclisten",
            "/ip4/127.0.0.1/tcp/{}".format(self.rpc_port),
            "-listen",
            "/ip4/127.0.0.1/tcp/{}".format(self.p2p_port),
            "-level",
            self.level,
            "-chaincfg",
            self.chaincfg,
            "-genesistime",
            str(self.genesis_time),
            "-colors",
        ]

        if len(self.connect) > 0:
            args += [
                "-connect",
                ",".join(self.connect)
            ]

        return args


def create_config_file(file_name):
    with open(tester.get_phore_path("regtest.json"), "r") as regtest_file:
        regtest_json = regtest_file.read()
    with open(file_name, "w") as file_to_write:
        file_to_write.write(regtest_json)


def snake_to_camel(snake_case: str) -> str:
    components = snake_case.split('_')
    return ''.join([c.title() for c in components])

class BeaconNode:
    TYPE = "beacon"

    def __init__(self, config: BeaconConfig):
        self._config = config
        self._process = None
        self.started = False
        self._rpc = None

        os.makedirs(self._config.datadir)

    def start(self):
        logging.info("Starting beacon node {}".format(self._config.index))

        config_file_name = os.path.join(self._config.datadir, 'testconfig.json')
        create_config_file(config_file_name)

        self._config.chaincfg = config_file_name

        self._process = process.Process(
            "beacon {}".format(self._config.index),
            self._config.beacon_executable,
            *self._config.get_args())

        self._process.start()

        self.started = True

    def stop(self):
        if self.started:
            self._process.signal_stop()
            self._process.join()

    def get_config(self) -> BeaconConfig:
        return self._config

    def wait_for_rpc(self):
        chan = grpc.insecure_channel("127.0.0.1:{}".format(self._config.rpc_port))

        ready = grpc.channel_ready_future(chan)
        while not ready.done():
            def test(conn):
                pass

            chan.subscribe(test, True)

            time.sleep(1)

            self._rpc = beaconrpc_pb2_grpc.BlockchainRPCStub(chan)

    _rpc: beaconrpc_pb2_grpc.BlockchainRPCStub

    def wait_for_slot(self, slot_to_wait: int):
        while True:
            slot_number_response: beaconrpc_pb2.SlotNumberResponse = self._rpc.GetSlotNumber(empty_pb2.Empty())
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
