package main

import (
	"flag"
	"os"
	"strconv"

	"GoPalette/internal/conf"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(
			env.NewSource(),
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	// Kratos env source keeps raw keys (e.g. DATA_DATABASE_SOURCE) and does not
	// automatically map underscore keys to nested fields like data.database.source.
	// Apply explicit overrides so .env (via direnv) reliably takes effect.
	if v := os.Getenv("DATA_DATABASE_DRIVER"); v != "" {
		bc.Data.Database.Driver = v
	}
	if v := os.Getenv("DATA_DATABASE_SOURCE"); v != "" {
		bc.Data.Database.Source = v
	}
	if v := os.Getenv("DATA_DATABASE_REDIS_ADDR"); v != "" {
		bc.Data.Redis.Addr = v
	} else if v := os.Getenv("DATA_REDIS_ADDR"); v != "" {
		bc.Data.Redis.Addr = v
	}
	if v := os.Getenv("DATA_DATABASE_REDIS_PASSWORD"); v != "" {
		bc.Data.Redis.Password = v
	} else if v := os.Getenv("DATA_REDIS_PASSWORD"); v != "" {
		bc.Data.Redis.Password = v
	}
	if v := os.Getenv("DATA_DATABASE_REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			bc.Data.Redis.Db = int32(db)
		}
	} else if v := os.Getenv("DATA_REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			bc.Data.Redis.Db = int32(db)
		}
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
