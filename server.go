package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fgrosse/fritz-mon/fritzbox"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	Logger    *zap.Logger
	Metrics   *Metrics
	Config    Config
	FritzBox  *fritzbox.Client
	interrupt chan os.Signal
}

var ErrServerClosed = fmt.Errorf("server closed")

func NewServer(conf Config, logger *zap.Logger) (*Server, error) {
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	client, err := fritzbox.New(conf.FritzBox.BaseURL, conf.FritzBox.Username, conf.FritzBox.Password, logger)
	if err != nil {
		return nil, fmt.Errorf("bad FRITZ!Box configuration")
	}

	return &Server{
		Logger:    logger,
		Metrics:   NewMetrics(logger),
		Config:    conf,
		FritzBox:  client,
		interrupt: interrupt,
	}, nil
}

func (s *Server) RegisterMetrics(r prometheus.Registerer) error {
	return s.Metrics.Register(r)
}

func (s *Server) Run() error {
	s.Logger.Info("Starting FRITZ!Box monitoring server",
		zap.String("listen_addr", s.Config.ListenAddr),
		zap.String("fritzbox", s.Config.FritzBox.BaseURL),
		zap.Duration("monitoring_interval", s.Config.MonitoringInterval),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler()) // TODO: use passed registry

	httpServer := &http.Server{
		Addr:    s.Config.ListenAddr,
		Handler: mux,
		// TODO: ErrorLog: log.NewStdLog(logger.Named("http")),
	}

	httpServerErr := make(chan error, 1)
	go func() {
		httpServerErr <- httpServer.ListenAndServe()
	}()

	ti := time.NewTicker(s.Config.MonitoringInterval)
	defer ti.Stop()

	// The first fetch should happen shortly after we started our HTTP server.
	firstFetch := time.After(time.Second)

	for {
		select {
		case <-firstFetch:
			firstFetch = nil // disable this case
			err := s.Metrics.FetchFrom(s.FritzBox)
			if err != nil {
				s.Logger.Error("Failed to fetch metrics", zap.Error(err))
			}

		case <-ti.C:
			err := s.Metrics.FetchFrom(s.FritzBox)
			if err != nil {
				s.Logger.Error("Failed to fetch metrics", zap.Error(err))
			}

		case sig := <-s.interrupt:
			s.Logger.Info("Shutting down server due to system interrupt",
				zap.Stringer("signal", sig),
			)

			err := s.FritzBox.Close()
			if err != nil {
				s.Logger.Error("Failed to close FRITZ!Box client", zap.Error(err))
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err = httpServer.Shutdown(ctx)
			cancel() // make sure the context never leaks past this point
			if err != nil {
				s.Logger.Error("Failed to shutdown HTTP server gracefully", zap.Error(err))
			}

			return ErrServerClosed

		case serverErr := <-httpServerErr:
			s.Logger.Error("HTTP server failed",
				zap.Error(serverErr),
				zap.String("listen_addr", s.Config.ListenAddr),
			)

			// Close everything just to be on the safe side. We don't care for
			// any other errors at this point.
			_ = httpServer.Close()
			_ = s.FritzBox.Close()

			return serverErr
		}
	}
}
