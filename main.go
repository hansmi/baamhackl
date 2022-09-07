package main

import (
	"context"
	crand "crypto/rand"
	"flag"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"

	"github.com/google/subcommands"
	"github.com/hansmi/baamhackl/move"
	"github.com/hansmi/baamhackl/selftest"
	"github.com/hansmi/baamhackl/sendfilechanges"
	"github.com/hansmi/baamhackl/watch"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// buildLogger configures a zap logger writing to, by default, standard error.
// Explicit output paths can be configured (see zap documentation).
func buildLogger(level zapcore.Level, outputs []string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Sampling = nil
	cfg.DisableCaller = true
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	cfg.EncoderConfig.EncodeName = func(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString("[" + loggerName + "]")
	}
	cfg.EncoderConfig.NewReflectedEncoder = nil
	cfg.Encoding = "console"
	if outputs != nil {
		cfg.OutputPaths = outputs
	}
	cfg.Level.SetLevel(level)

	return cfg.Build()
}

// seedPseudoRand initializes the pseudo random number generator from
// a cryptographically secure source.
func seedPseudoRand() {
	if num, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64)); err == nil {
		rand.Seed(num.Int64())
	} else {
		panic("Seeding random number generator failed: " + err.Error())
	}
}

func main() {
	seedPseudoRand()

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&watch.Command{}, "")
	subcommands.Register(&move.IntoCommand{}, "")
	subcommands.Register(&selftest.Command{}, "")

	subcommands.Register(&sendfilechanges.Command{}, "internal")

	logLevel := zap.LevelFlag("log_level", zap.InfoLevel, "Log level for stderr.")

	flag.Parse()

	logger, err := buildLogger(*logLevel, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initializing logger failed: %v\n", err)
		os.Exit(int(subcommands.ExitFailure))
	}

	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)

	os.Exit(int(subcommands.Execute(context.Background())))
}
