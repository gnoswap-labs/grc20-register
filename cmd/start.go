package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"

	"github.com/gnoswap-labs/grc20-register/client"
	"github.com/gnoswap-labs/grc20-register/config"
	"github.com/gnoswap-labs/grc20-register/events"
	"github.com/gnoswap-labs/grc20-register/fetch"
	"github.com/gnoswap-labs/grc20-register/storage"
)

const (
	configFlagName = "config"
	// envPrefix      = "GNO_REGISTER"
)

const (
	defaultRemote  = "http://127.0.0.1:26657"
	defaultChainId = "dev"

	defaultGasFee    = "1000000ugnot"
	defaultGasWanted = "10000000"

	defaultDBPath = "register-db"
)

// addPkgCfg wraps the addPkg
// root command configuration
type addPkgCfg struct {
	config *config.Config

	remote    string
	chainId   string
	gasFee    string
	gasWanted string

	dbPath   string
	logLevel string

	maxSlots     int
	maxChunkSize int64
}

// newStartCmd creates the register start command
func newStartCmd() *ffcli.Command {
	cfg := &addPkgCfg{}

	fs := flag.NewFlagSet("start", flag.ExitOnError)
	cfg.registerRootFlags(fs)

	return &ffcli.Command{
		Name:       "start",
		ShortUsage: "start [flags]",
		ShortHelp:  "Starts the grc20 register service",
		LongHelp:   "Starts the grc20 register service, which includes the fetcher",
		FlagSet:    fs,
		Exec: func(ctx context.Context, _ []string) error {
			return cfg.exec(ctx)
		},
	}
}

// registerFlags registers the register start command flags
func (c *addPkgCfg) registerRootFlags(fs *flag.FlagSet) {
	// Config flag
	fs.String(
		configFlagName,
		"",
		"the path to the command configuration file [TOML]",
	)

	// Top level flags
	fs.StringVar(
		&c.dbPath,
		"db-path",
		defaultDBPath,
		"the absolute path for the register DB (embedded)",
	)

	fs.StringVar(
		&c.logLevel,
		"log-level",
		zap.InfoLevel.String(),
		"the log level for the CLI output",
	)

	fs.IntVar(
		&c.maxSlots,
		"max-slots",
		fetch.DefaultMaxSlots,
		"the amount of slots (workers) the fetcher employs",
	)

	fs.Int64Var(
		&c.maxChunkSize,
		"max-chunk-size",
		fetch.DefaultMaxChunkSize,
		"the range for fetching blockchain data by a single worker",
	)

	fs.StringVar(
		&c.gasFee,
		"gas-fee",
		defaultGasFee,
		"the static gas fee for the transaction. Format: <AMOUNT>ugnot",
	)

	fs.StringVar(
		&c.gasWanted,
		"gas-wanted",
		defaultGasWanted,
		"the static gas wanted for the transaction. Format: <AMOUNT>ugnot",
	)

	fs.StringVar(
		&c.remote,
		"remote",
		defaultRemote,
		"the JSON-RPC URL of the Gno chain",
	)

	fs.StringVar(
		&c.chainId,
		"chain-id",
		defaultChainId,
		"the chainId of the Gno chain",
	)
}

// exec executes the register start command
func (c *addPkgCfg) exec(ctx context.Context) error {
	// Parse the log level
	logLevel, err := zap.ParseAtomicLevel(c.logLevel)
	if err != nil {
		return fmt.Errorf("unable to parse log level, %w", err)
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.Level = logLevel

	// Create a new logger
	logger, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("unable to create logger, %w", err)
	}

	// Create a DB instance
	db, err := storage.NewPebble(c.dbPath)
	if err != nil {
		return fmt.Errorf("unable to open storage DB, %w", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("unable to gracefully close DB", zap.Error(err))
		}
	}()

	// Create an Event Manager instance
	em := events.NewManager()

	// Create a TM2 client
	tm2Client, err := client.NewClient(c.remote)
	if err != nil {
		return fmt.Errorf("unable to create client, %w", err)
	}

	// Create the fetcher service
	f := fetch.New(
		db,
		tm2Client,
		em,
		fetch.WithLogger(
			logger.Named("fetcher"),
		),
		fetch.WithMaxSlots(c.maxSlots),
		fetch.WithMaxChunkSize(c.maxChunkSize),
	)

	// Create a new waiter
	w := newWaiter(ctx)

	// Add the fetcher service
	w.add(f.FetchChainData)

	// Wait for the services to stop
	return errors.Join(
		w.wait(),
		logger.Sync(),
	)
}
