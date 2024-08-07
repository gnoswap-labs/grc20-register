package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"

	"github.com/gnolang/tx-indexer/client"
	"github.com/gnolang/tx-indexer/events"
	"github.com/gnolang/tx-indexer/fetch"
	"github.com/gnolang/tx-indexer/serve"
	"github.com/gnolang/tx-indexer/serve/graph"
	"github.com/gnolang/tx-indexer/storage"

	rpcClient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
)

const (
	defaultRemote  = "http://127.0.0.1:26657"
	defaultChainId = "dev"
	defaultDBPath  = "register-db"
)

type startCfg struct {
	listenAddress string
	remote        string
	chainId       string
	dbPath        string
	logLevel      string

	maxSlots     int
	maxChunkSize int64

	rateLimit int
}

// newStartCmd creates the indexer start command
func newStartCmd() *ffcli.Command {
	cfg := &startCfg{}

	fs := flag.NewFlagSet("start", flag.ExitOnError)
	cfg.registerFlags(fs)

	return &ffcli.Command{
		Name:       "start",
		ShortUsage: "start [flags]",
		ShortHelp:  "Starts the indexer service",
		LongHelp:   "Starts the indexer service, which includes the fetcher and JSON-RPC server",
		FlagSet:    fs,
		Exec: func(ctx context.Context, _ []string) error {
			return cfg.exec(ctx)
		},
	}
}

// registerFlags registers the indexer start command flags
func (c *startCfg) registerFlags(fs *flag.FlagSet) {
	fs.StringVar(
		&c.listenAddress,
		"listen-address",
		serve.DefaultListenAddress,
		"the IP:PORT URL for the indexer JSON-RPC server",
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
		"the chain-id of Gno chain",
	)

	fs.StringVar(
		&c.dbPath,
		"db-path",
		defaultDBPath,
		"the absolute path for the indexer DB (embedded)",
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

	fs.IntVar(
		&c.rateLimit,
		"http-rate-limit",
		0,
		"the maximum HTTP requests allowed per minute per IP, unlimited by default",
	)
}

// exec executes the indexer start command
func (c *startCfg) exec(ctx context.Context) error {
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
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("unable to gracefully close DB", zap.Error(closeErr))
		}
	}()

	// Create an Event Manager instance
	em := events.NewManager()

	// Create a TM2 client
	tm2Client, err := client.NewClient(c.remote)
	if err != nil {
		return fmt.Errorf("unable to create client, %w", err)
	}

	// Create a TM2 RPC client
	rpcClient, err := rpcClient.NewHTTPClient(c.remote)
	if err != nil {
		return fmt.Errorf("unable to create rpc client, %w", err)
	}

	// Create the fetcher service
	f := fetch.New(
		db,
		tm2Client,
		*rpcClient,
		em,
		fetch.WithLogger(
			logger.Named("fetcher"),
		),
		fetch.WithMaxSlots(c.maxSlots),
		fetch.WithMaxChunkSize(c.maxChunkSize),
	)

	// Create the JSON-RPC service
	j := setupJSONRPC(
		db,
		em,
		logger,
	)

	mux := chi.NewMux()

	mux.Use(NewCORSHandler())

	mux = j.SetupRoutes(mux)
	mux = graph.Setup(db, em, mux)

	// Create the HTTP server
	hs := serve.NewHTTPServer(mux, c.listenAddress, logger.Named("http-server"))

	// Create a new waiter
	w := newWaiter(ctx)

	// Add the fetcher service
	w.add(f.FetchChainData)

	// Add the JSON-RPC service
	w.add(hs.Serve)

	// Wait for the services to stop
	return errors.Join(
		w.wait(),
		logger.Sync(),
	)
}

// setupJSONRPC sets up the JSONRPC instance
func setupJSONRPC(
	db *storage.Pebble,
	em *events.Manager,
	logger *zap.Logger,
) *serve.JSONRPC {
	j := serve.NewJSONRPC(
		em,
		serve.WithLogger(
			logger.Named("json-rpc"),
		),
	)

	// Transaction handlers
	j.RegisterTxEndpoints(db)

	// Block handlers
	j.RegisterBlockEndpoints(db)

	// Sub handlers
	j.RegisterSubEndpoints(db)

	return j
}

func NewCORSHandler() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(writer http.ResponseWriter, request *http.Request) {
				allowedHeaders := "*"

				if origin := request.Header.Get("Origin"); origin != "" {
					writer.Header().Set("Access-Control-Allow-Origin", "*")
					writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
					writer.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				}

				next.ServeHTTP(writer, request)
			},
		)
	}
}
