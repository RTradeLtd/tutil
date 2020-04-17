package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/RTradeLtd/cmd/v2"
	"github.com/RTradeLtd/config/v2"
	"github.com/RTradeLtd/database/v2"
	"github.com/RTradeLtd/database/v2/models"
	useremailmigration "github.com/RTradeLtd/tutil/migrations/user"
	"github.com/RTradeLtd/tutil/pin"
	"github.com/jinzhu/gorm"
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
	user           *string
	// bucket flags
	bucketLocation *string
	accountTier    *string
	credits        *float64
	gcOutFile      *string

	notifyDays      *int
	expireFrequency *time.Duration
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
	emailRecipient = f.String("email-recipient", "alext@rtradetechnologies.com",
		"email to send metrics to")
	recipientName = f.String("recipient-name", "",
		"email recipient name")

	user = f.String("user", "", "user to operate commands against")

	accountTier = f.String("account.tier", "", "accoutn tier to apply")

	credits = f.Float64("credits", 0, "the amount of credits to add")

	gcOutFile = f.String("gc.out.file", fmt.Sprintf(
		"collected_garbage-%v.txt", time.Now().UnixNano()),
		"the destination file to store garbage collected records in",
	)
	expireFrequency = flag.Duration("pin.expire.frequency", time.Hour, "enables controlling the frequency of pin expiration")
	notifyDays = f.Int("notify.days", 7, "the number of days before we will warn about an expired pin")
	return f
}

func newDB(cfg *config.TemporalConfig, noSSL bool) (*gorm.DB, error) {
	dbm, err := database.New(cfg, database.Options{
		SSLModeDisable: noSSL,
	})
	if err != nil {
		return nil, err
	}
	return dbm.DB, nil
}

var commands = map[string]cmd.Cmd{
	"migrations": {
		Blurb:         "manage complex database migrations",
		ChildRequired: true,
		Children: map[string]cmd.Cmd{
			"unverified-user-migration": {
				Blurb: "verify all unverified user accounts for apr 17 migration",
				Action: func(cfg config.TemporalConfig, flags map[string]string) {
					db, err := newDB(&cfg, *dbNoSSL)
					if err != nil {
						log.Fatal(err)
					}
					userm := useremailmigration.NewUserMigration(db)
					count, err := userm.VerifyUnverifiedUsers()
					if err != nil {
						log.Fatal(err)
					}
					log.Printf("verified %v users", count)
				},
			},
		},
	},
	"reset": {
		Blurb: "reset user account tier",
		Action: func(cfg config.TemporalConfig, flags map[string]string) {
			if *user == "" {
				log.Fatal("user flag not specified")
			}
			db, err := newDB(&cfg, *dbNoSSL)
			if err != nil {
				log.Fatal(err)
			}
			usage := models.NewUsageManager(db)
			if err := usage.UpdateTier(*user, models.Free); err != nil {
				log.Fatal(err)
			}
		},
	},
	"pin-expire-service": {
		Blurb:       "runs pin garbage collection service",
		Description: "regularly removes pins from the system, and saves the removes ones to disk. Note that this doesn't actually remove it from our servers",
		Action: func(cfg config.TemporalConfig, flags map[string]string) {
			db, err := newDB(&cfg, *dbNoSSL)
			if err != nil {
				log.Fatal(err)
			}
			pinUtil, err := pin.NewPinUtil(db, &cfg)
			if err != nil {
				log.Fatal(err)
			}
			totalRemoved, err := pinUtil.PinExpirationService(
				ctx, *expireFrequency,
			)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("removed %v pins", totalRemoved)
		},
	},
	"pin-notifiers": {
		Blurb:       "pin expiration notifier",
		Description: "warns users when their pins are reaching their expiration date",
		Action: func(cfg config.TemporalConfig, flags map[string]string) {
			db, err := newDB(&cfg, *dbNoSSL)
			if err != nil {
				log.Fatal(err)
			}
			pinUtil, err := pin.NewPinUtil(db, &cfg)
			if err != nil {
				log.Fatal(err)
			}
			messages, err := pinUtil.GetPinsToRemind(*notifyDays)
			if err != nil {
				log.Fatal(err)
			}
			for _, message := range messages {
				email := ""
				// enable debugging by sending messages to rtrade instead
				// otherwise use the users email
				if *debug {
					email = *emailRecipient
				} else {
					email = message.EmailAddress
				}
				_, err := pinUtil.Mail.SendEmail(
					"Temporal: You Have Pins About To Expire",
					message.Message,
					"text/html",
					message.UserName,
					email,
				)
				if err != nil {
					log.Printf(
						"error: failed to send message to %s with err %s",
						message.EmailAddress, err.Error(),
					)
					continue
				}
			}

		},
	},
	"gc": {
		Blurb:         "manage garbage collection",
		Description:   "Allows managing garbage collection of Temporal",
		ChildRequired: true,
		Children: map[string]cmd.Cmd{
			"run": cmd.Cmd{
				Blurb:       "run a pin garbage collection",
				Description: "parse uploads and collect expired pins",
				Action: func(cfg config.TemporalConfig, flags map[string]string) {
					db, err := newDB(&cfg, *dbNoSSL)
					if err != nil {
						log.Fatal(err)
					}
					pinUtil, err := pin.NewPinUtil(db, &cfg)
					if err != nil {
						log.Fatal(err)
					}
					expiredPins, err := pinUtil.GetExpiredPins()
					if err != nil {
						log.Fatal(err)
					}
					if err := pinUtil.ExpirePins(expiredPins); err != nil {
						log.Fatal(err)
					}
					var formattedOutput string
					for _, pin := range expiredPins {
						formattedOutput = fmt.Sprintf("%s\n%+v\n", formattedOutput, pin)
					}
					if err := ioutil.WriteFile(
						*gcOutFile,
						[]byte(formattedOutput),
						os.FileMode(0640),
					); err != nil {
						log.Fatal(err)
					}
				},
			},
			"run-dry": {
				Blurb:       "run a dry pin garbage collection",
				Description: "runs a dry run of the garbage collection period",
				Action: func(cfg config.TemporalConfig, flags map[string]string) {
					db, err := newDB(&cfg, *dbNoSSL)
					if err != nil {
						log.Fatal(err)
					}
					pinUtil, err := pin.NewPinUtil(db, &cfg)
					if err != nil {
						log.Fatal(err)
					}
					expiredPins, err := pinUtil.GetExpiredPins()
					if err != nil {
						log.Fatal(err)
					}
					var formattedOutput string
					for _, pin := range expiredPins {
						formattedOutput = fmt.Sprintf("%s\n%+v\n", formattedOutput, pin)
					}
					if err := ioutil.WriteFile(
						*gcOutFile,
						[]byte(formattedOutput),
						os.FileMode(0640),
					); err != nil {
						log.Fatal(err)
					}
				},
			},
		},
	},
	"upgrade-tier": {
		Blurb:       "upgrade account tier",
		Description: "used to perform an account tier upgrade",
		Action: func(cfg config.TemporalConfig, flags map[string]string) {
			if *user == "" {
				log.Fatal("user flag is empty")
			}
			db, err := newDB(&cfg, *dbNoSSL)
			if err != nil {
				log.Fatal(err)
			}
			usg := models.NewUsageManager(db)
			switch models.DataUsageTier(*accountTier).PricePerGB() {
			case 0.07, 0.05, 9999:
				break
			default:
				log.Fatal("invalid account tier")
			}
			if err := usg.UpdateTier(*user, models.DataUsageTier(*accountTier)); err != nil {
				log.Fatal("failed to upgrade tier", err)
			}
		},
	},
	"add-credits": {
		Blurb:       "add credits to an account",
		Description: "used to increase the credits balance of an account",
		Action: func(cfg config.TemporalConfig, flags map[string]string) {
			if *user == "" {
				log.Fatal("user flag is empty")
			}
			if *credits == 0 {
				log.Fatal("credits flag is empty")
			}
			db, err := newDB(&cfg, *dbNoSSL)
			if err != nil {
				log.Fatal(err)
			}
			um := models.NewUserManager(db)
			_, err = um.AddCredits(*user, *credits)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("credits granted")
		},
	},
}

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
