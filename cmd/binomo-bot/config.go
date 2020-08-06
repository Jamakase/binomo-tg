package main

import (
	"github.com/awesomeProject/internal/app/binomo"
	"github.com/awesomeProject/internal/app/bot"
	"github.com/awesomeProject/internal/platform/log"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	// Log configuration
	Log log.Config
	// App configuration
	App appConfig
}

// Process post-processes configuration after loading it.
func (configuration) Process() error {
	return nil
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	if err := c.App.Validate(); err != nil {
		return err
	}

	if err := c.Log.Validate(); err != nil {
		return err
	}

	return nil
}

// appConfig represents the application related configuration.
type appConfig struct {
	Bot bot.Config
	Binomo binomo.Config
}

// Validate validates the configuration.
func (c appConfig) Validate() error {
	if err := c.Bot.Validate(); err != nil {
		return err
	}

	return nil
}

// configure configures some defaults in the Viper instance.
func configure(v *viper.Viper, p *pflag.FlagSet) {
	// Viper settings
	v.AddConfigPath(".")
	v.AddConfigPath("$CONFIG_DIR/")

	// Environment variable settings
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Global configuration
	v.SetDefault("shutdownTimeout", 15*time.Second)
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		v.SetDefault("no_color", true)
	}

	// Log configuration
	v.SetDefault("log.format", "json")
	v.SetDefault("log.level", "info")
	v.RegisterAlias("log.noColor", "no_color")
}
