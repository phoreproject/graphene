package main

import (
	"github.com/phoreproject/graphene/cfg"
	"github.com/phoreproject/graphene/validator/config"

	"github.com/phoreproject/graphene/utils"
	"github.com/phoreproject/graphene/validator/module"

	logger "github.com/sirupsen/logrus"
)

func main() {
	validatorConfig := config.Options{}
	globalConfig := cfg.GlobalOptions{}
	err := cfg.LoadFlags(&validatorConfig, &globalConfig)
	if err != nil {
		logger.Fatal(err)
	}

	logger.StandardLogger().SetFormatter(&logger.TextFormatter{
		ForceColors: globalConfig.ForceColors,
	})

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
