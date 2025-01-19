package main

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type Runnable struct {
	path      string
	isRunning atomic.Bool
	cancel    context.CancelFunc
}

func (r *Runnable) IsRunning() bool {
	return r.isRunning.Load()
}

func NewRunnable(path string) *Runnable {
	return &Runnable{path: path}
}

func (r *Runnable) Start(ctx context.Context, f func(context.Context, string) error) bool {
	if r.IsRunning() {
		return false
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.isRunning.Store(true)

	go func() {
		defer r.isRunning.Store(false)
		if err := f(ctx, r.path); err != nil {
			log.Error().Err(err).Msg("stopped app")
		}
	}()

	return true
}

func (r *Runnable) Stop() {
	if !r.IsRunning() {
		return
	}

	r.cancel()
}

func (r *Runnable) WaitForStop() {
	for r.IsRunning() {
		time.Sleep(time.Millisecond * 100)
	}
}
