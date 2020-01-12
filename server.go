package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
	)

	if s.Logger.Check(zap.DebugLevel, "") == nil {
		s.Logger.Info("If you want to see more verbose log run with -debug")
	} else {
		s.Logger.Debug("Debug logging is enabled")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	httpServer := &http.Server{
		Addr:    s.Config.ListenAddr,
		Handler: mux,
	}

	ctx, shutdown := context.WithCancel(context.Background())

	var serverErr error
	go func() {
		err := httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			serverErr = fmt.Errorf("HTTP server failed: %w", err)
		}
		shutdown()
	}()

	go func() {
		select {
		case sig := <-s.interrupt:
			s.Logger.Info("Shutting down server due to system interrupt",
				zap.Stringer("signal", sig),
			)
			shutdown()
		case <-ctx.Done():
			return
		}
	}()

	s.CollectMetrics(ctx)

	err := s.FritzBox.Close()
	if err != nil {
		s.Logger.Error("Failed to close FRITZ!Box client", zap.Error(err))
	}

	s.Logger.Info("HTTP Server is shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err = httpServer.Shutdown(ctx)
	cancel() // make sure the context never leaks past this point
	if err != nil {
		s.Logger.Error("Failed to shutdown HTTP server gracefully", zap.Error(err))
	}

	return serverErr
}

func (s *Server) CollectMetrics(ctx context.Context) {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go s.deviceMetricsLoop(ctx, wg, s.Config.DeviceMonitoringInterval)
	go s.networkMetricsLoop(ctx, wg, s.Config.NetworkMonitoringInterval)
	wg.Wait()
}

func newTicker(ctx context.Context, interval time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- time.Now() // trigger first metrics collection immediately

	go func() {
		ti := time.NewTicker(interval)
		defer ti.Stop()

		for {
			var next time.Time
			select {
			case next = <-ti.C:
			case <-ctx.Done():
				return
			}

			select {
			case ch <- next:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

func (s *Server) deviceMetricsLoop(ctx context.Context, wg *sync.WaitGroup, interval time.Duration) {
	s.Logger.Info("Monitoring device metrics", zap.Duration("interval", interval))

	ticker := newTicker(ctx, interval)
	for {
		select {
		case <-ticker:
			err := s.Metrics.Devices.FetchFrom(ctx, s.FritzBox)
			if err != nil && !errors.Is(err, context.Canceled) {
				s.Logger.Error("Failed to fetch device metrics", zap.Error(err))
			}

		case <-ctx.Done():
			s.Logger.Info("Device monitoring stopped")
			wg.Done()
			return
		}
	}
}

func (s *Server) networkMetricsLoop(ctx context.Context, wg *sync.WaitGroup, interval time.Duration) {
	s.Logger.Info("Monitoring network metrics", zap.Duration("interval", interval))

	ticker := newTicker(ctx, interval)
	// TODO: actually we fetch the last 20 5 second buckets so we want to leverage that somehow

	for {
		select {
		case <-ctx.Done():
			s.Logger.Info("Network monitoring stopped")
			wg.Done()
			return

		case <-ticker:
			err := s.Metrics.Network.FetchFrom(ctx, s.FritzBox)
			if err != nil && !errors.Is(err, context.Canceled) {
				s.Logger.Error("Failed to fetch network metrics", zap.Error(err))
			}
		}
	}
}
