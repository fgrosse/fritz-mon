package main

import (
	"flag"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	setup := flag.Bool("setup", false, "setup configuration file interactively")
	verbose := flag.Bool("debug", false, "enable verbose log output")
	config := flag.String("config", "fritz-mon.yml", "path to the configuration file")
	flag.Parse()

	if *setup {
		runSetup()
		return
	}

	logger := newLogger(*verbose)
	defer func() { _ = logger.Sync() }()

	conf, err := LoadConfiguration(*config, logger)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	server, err := NewServer(conf, logger)
	if err != nil {
		logger.Fatal("Failed to create new server", zap.Error(err))
	}

	err = server.RegisterMetrics(prometheus.DefaultRegisterer)
	if err != nil {
		logger.Fatal("Failed to register server metrics", zap.Error(err))
	}

	err = server.Run()
	if err != nil && err != ErrServerClosed {
		logger.Fatal("Fatal server error", zap.Error(err))
	}

	logger.Info(`Shutdown complete. Have a nice day  \ʕ◔ϖ◔ʔ/`)
}

func newLogger(verbose bool) *zap.Logger {
	level := zap.InfoLevel
	if verbose {
		level = zap.DebugLevel
	}

	cfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: false,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			NameKey:        "N",
			MessageKey:     "M",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return logger
}
