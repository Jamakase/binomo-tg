package main

import (
	"github.com/awesomeProject/internal/app/binomo"
	"github.com/awesomeProject/internal/app/bot"
	"github.com/awesomeProject/internal/platform/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corlog "log"
	"os"
)

const (
	// appName is an identifier-like name used anywhere this app needs to be identified.
	//
	// It identifies the application itself, the actual instance needs to be identified via environment
	// and other details.
	appName = "mga"

	// friendlyAppName is the visible name of the application.
	friendlyAppName = "Modern Go Application"
)

func main() {
	v, p := viper.New(), pflag.NewFlagSet(friendlyAppName, pflag.ExitOnError)

	configure(v, p)
	err := v.ReadInConfig()

	_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
	if configFileNotFound {
		corlog.Panic(err)
	}

	var config configuration
	err = v.Unmarshal(&config)
	//emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	if err != nil {
		corlog.Panic(err)
	}

	// Create logger (first thing after configuration loading)
	logger := log.NewLogger(config.Log)

	// Override the global standard library logger to make sure everything uses our logger
	log.SetStandardLogger(logger)

	if configFileNotFound {
		logger.Warn("configuration file not found")
	}

	err = config.Validate()
	if err != nil {
		logger.Error(err.Error())

		os.Exit(3)
	}

	binomo := binomo.New(config.App.Binomo)

	bot.New(config.App.Bot, binomo)
}
