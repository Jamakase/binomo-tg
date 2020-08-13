package main

import (
	"context"
	"emperror.dev/emperror"
	"emperror.dev/errors"
	"fmt"
	"github.com/awesomeProject/internal/app/binomo"
	"github.com/awesomeProject/internal/app/bot"
	"github.com/awesomeProject/internal/app/bot/job"
	"github.com/awesomeProject/internal/common/commonadapter"
	"github.com/awesomeProject/internal/platform/log"
	"github.com/sagikazarmark/appkit/buildinfo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

const (
	// appName is an identifier-like name used anywhere this app needs to be identified.
	//
	// It identifies the application itself, the actual instance needs to be identified via environment
	// and other details.
	//appName = "binomo-bot"

	// friendlyAppName is the visible name of the application.
	friendlyAppName = "Modern Go Application"
)

var (
	version    string
	commitHash string
	buildDate  string
)

func main() {
	v, p := viper.New(), pflag.NewFlagSet(friendlyAppName, pflag.ExitOnError)

	configure(v, p)

	p.String("config", "", "Configuration file")
	p.Bool("version", false, "Show version information")

	_ = p.Parse(os.Args[1:])

	if v, _ := p.GetBool("version"); v {
		fmt.Printf("%s version %s (%s) built on %s\n", friendlyAppName, version, commitHash, buildDate)

		os.Exit(0)
	}

	if c, _ := p.GetString("config"); c != "" {
		v.SetConfigFile(c)
	}

	err := v.ReadInConfig()
	_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
	if !configFileNotFound {
		emperror.Panic(errors.Wrap(err, "failed to read configuration"))
	}

	var config configuration
	err = v.Unmarshal(&config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	err = config.Process()
	emperror.Panic(errors.WithMessage(err, "failed to process configuration"))

	// Create logger (first thing after configuration loading)
	logger := log.NewLogger(config.Log)

	// Override the global standard library logger to make sure everything uses our logger
	log.SetStandardLogger(logger)

	if configFileNotFound {
		logger.Warn("configuration file not found")
	}

	if configFileNotFound {
		logger.Warn("configuration file not found")
	}

	err = config.Validate()
	if err != nil {
		logger.Error(err.Error())

		os.Exit(3)
	}

	// Configure error handler
	//errorHandler := logurhandler.New(logger)
	//defer emperror.HandleRecover(errorHandler)

	buildInfo := buildinfo.New(version, commitHash, buildDate)

	logger.Info("starting application", buildInfo.Fields())

	// Set client options
	clientOptions := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%d", config.Database.Host, config.Database.Port))
	clientOptions.SetAuth(options.Credential{
		Username: config.Database.Username,
		Password: config.Database.Password,
	})

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.TODO(), clientOptions)

	if err != nil {
		emperror.Panic(errors.Wrap(err, "failed to init mongodb"))
	}

	// Check the connection
	err = mongoClient.Ping(context.TODO(), nil)

	if err != nil {
		emperror.Panic(errors.Wrap(err, "failed to ping db"))
	}

	logger.Info("connected to db")

	{
		logger := commonadapter.NewLogger(logger)
		//errorHandler := emperror.WithFilter(
		//	emperror.WithContextExtractor(errorHandler, appkit.ContextExtractor),
		//	appkiterrors.IsServiceError, // filter out service errors
		//)

		db := mongoClient.Database("test")

		binomo := binomo.New(config.App.Binomo)
		jobStore := job.NewStore()
		msgRepo := bot.NewConfigRepo(db)
		userStateRepo := bot.NewRepo()

		bt := bot.New(logger, config.App.Bot, binomo, userStateRepo, jobStore, msgRepo)

		bt.Run(context.TODO())
	}
}
