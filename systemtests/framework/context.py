from . import util

import argparse
import os
import shutil

class Context :
    def __init__(
            self,
            directory = None,
            delete_data_on_exit = True,
        ) :
        self._directory = directory
        self._delete_data_on_exit = delete_data_on_exit

        self._has_initialized = False
        self._beacon_executable = None
        self._validator_executable = None

    def get_directory(self) :
        return self._directory
        
    def set_directory(self, directory) :
        self._directory = directory
        
    def should_delete_data_on_exit(self) :
        return self._delete_data_on_exit
        
    def get_beacon_executable(self) :
        self._check_initialize()
        return self._beacon_executable

    def get_validator_executable(self) :
        self._check_initialize()
        return self._validator_executable

    def _check_initialize(self) :
        if not self._has_initialized :
            self._parse_command_line()
            self._parse_env()
            self._check_default()
            self._has_initialized = True
            
    def _parse_command_line(self) :
        parser = argparse.ArgumentParser(description = 'Command line arguments')
        parser.add_argument(
            '--beacon',
            type = str,
            action = 'store',
            dest = 'beacon',
            help='Beacon executable'
        )
        parser.add_argument(
            '--validator',
            type = str,
            action = 'store',
            dest = 'validator',
            help='Validator executable'
        )
        args = parser.parse_args()
        self._beacon_executable = args.beacon
        self._validator_executable = args.validator
    
    def _parse_env(self) :
        if self._beacon_executable == None :
            self._beacon_executable = util.get_dict_value(os.environ, 'BEACON')
        if self._validator_executable == None :
            self._validator_executable = util.get_dict_value(os.environ, 'VALIDATOR')

    def _check_default(self) :
        if self._beacon_executable == None :
            self._beacon_executable = 'synapsebeacon'
            if shutil.which(self._beacon_executable) == None :
                self._beacon_executable = '../../synapsebeacon'

        if self._validator_executable == None :
            self._validator_executable = 'synapsevalidator'
            if shutil.which(self._validator_executable) == None :
                self._validator_executable = '../../synapsevalidator'
