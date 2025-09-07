package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

type Shutdown struct {
	callbacks []func(context.Context) error
	mu        sync.Mutex
	timeout   time.Duration
}

func New(timeout time.Duration) *Shutdown {
	return &Shutdown{
		callbacks: make([]func(context.Context) error, 0),
		timeout:   timeout,
	}
}

func (s *Shutdown) Add(callback func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, callback)
}

func (s *Shutdown) Wait() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Execute all callbacks
	s.mu.Lock()
	callbacks := make([]func(context.Context) error, len(s.callbacks))
	copy(callbacks, s.callbacks)
	s.mu.Unlock()

	var wg sync.WaitGroup
	for i, callback := range callbacks {
		wg.Add(1)
		go func(idx int, cb func(context.Context) error) {
			defer wg.Done()
			if err := cb(ctx); err != nil {
				log.Error().Err(err).Int("callback_index", idx).Msg("Error during shutdown")
			}
		}(i, callback)
	}

	// Wait for all callbacks to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Graceful shutdown completed")
	case <-ctx.Done():
		log.Warn().Msg("Shutdown timeout exceeded, forcing exit")
	}
}

// Global shutdown manager
var globalShutdown *Shutdown

func Init(timeout time.Duration) {
	globalShutdown = New(timeout)
}

func Add(callback func(context.Context) error) {
	if globalShutdown != nil {
		globalShutdown.Add(callback)
	}
}

func Wait() {
	if globalShutdown != nil {
		globalShutdown.Wait()
	}
}
