package main

import (
	"context"
	"flag"
	"os"

	"github.com/RTradeLtd/gorm"

	"github.com/RTradeLtd/cmd"
	"github.com/RTradeLtd/config"
	"github.com/RTradeLtd/database"
)

// Version denotes the tag of this build
var Version string

// globals
var (
	ctx    context.Context
	cancel context.CancelFunc
)

// command-line flags
var (
	devMode        *bool
	debug          *bool
	configPath     *string
	dbNoSSL        *bool
	dbMigrate      *bool
	sendEmail      *bool
	emailRecipient *string
	recipientName  *string
	uploadType     *string
	// bucket flags
	bucketLocation *string
)

func baseFlagSet() *flag.FlagSet {
	var f = flag.NewFlagSet("", flag.ExitOnError)

	// basic flags
	devMode = f.Bool("dev", false,
		"toggle dev mode")
	debug = f.Bool("debug", false,
		"toggle debug mode")
	configPath = f.String("config", os.Getenv("CONFIG_DAG"),
		"path to Temporal configuration")
	uploadType = f.String("upload-type", "file",
		"type of uploads to query against")

	// db configuration
	dbNoSSL = f.Bool("db.no_ssl", false,
		"toggle SSL connection with database")
	dbMigrate = f.Bool("db.migrate", false,
		"toggle whether a database migration should occur")

	// email flags
	sendEmail = f.Bool("email-enabled", false,
		"used to activate email notification")
	emailRecipient = f.String("email-recipient", "",
		"email to send metrics to")
	recipientName = f.String("recipient-name", "",
		"email recipient name")

	return f
}

func newDB(cfg config.TemporalConfig, noSSL bool) (*gorm.DB, error) {
	return database.OpenDBConnection(database.DBOptions{
		User:           cfg.Database.Username,
		Password:       cfg.Database.Password,
		Address:        cfg.Database.URL,
		Port:           cfg.Database.Port,
		SSLModeDisable: noSSL,
	})
}

var commands = map[string]cmd.Cmd{}

func main() {
	if Version == "" {
		Version = "latest"
	}

	// initialize global context
	ctx, cancel = context.WithCancel(context.Background())

	// create app
	tutil := cmd.New(commands, cmd.Config{
		Name:     "Temporal Utility",
		ExecName: "tutil",
		Version:  Version,
		Desc:     "Temporal command line utility client",
		Options:  baseFlagSet(),
	})

	// run no-config commands, exit if command was run
	if exit := tutil.PreRun(nil, os.Args[1:]); exit == cmd.CodeOK {
		os.Exit(0)
	}

	// load config
	tCfg, err := config.LoadConfig(*configPath)
	if err != nil {
		println("failed to load config at", *configPath)
		os.Exit(1)
	}

	// load arguments
	flags := map[string]string{
		"dbPass":  tCfg.Database.Password,
		"dbURL":   tCfg.Database.URL,
		"dbUser":  tCfg.Database.Username,
		"version": Version,
	}

	// execute
	os.Exit(tutil.Run(*tCfg, flags, os.Args[1:]))
}
