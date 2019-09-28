package main

import (
	"github.com/phoreproject/synapse/cfg"
	"github.com/phoreproject/synapse/validator/config"

	"github.com/phoreproject/synapse/utils"
	"github.com/phoreproject/synapse/validator/module"

	logger "github.com/sirupsen/logrus"
)

func main() {
	validatorConfig := config.Options{}
	globalConfig := cfg.GlobalOptions{}
	err := cfg.LoadFlags(&validatorConfig, &globalConfig)
	if err != nil {
		logger.Fatal(err)
	}

	utils.CheckNTP()

	a, err := module.NewValidatorApp(validatorConfig)
	if err != nil {
		logger.Fatal(err)
	}

	err = a.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
