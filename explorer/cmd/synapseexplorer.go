package main

import (
	"github.com/phoreproject/synapse/utils"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/phoreproject/synapse/explorer"

	logger "github.com/sirupsen/logrus"
)

func main() {
	utils.CheckNTP()

	config := explorer.LoadConfig()
	changed, newLimit, err := utils.ManageFdLimit()
	if err != nil {
		panic(err)
	}
	if changed {
		logger.Infof("changed ulimit to: %d", newLimit)
	}

	ex, err := explorer.NewExplorer(config)
	if err != nil {
		panic(err)
	}

	err = ex.StartExplorer()
	if err != nil {
		panic(err)
	}
}
