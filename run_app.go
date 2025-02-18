package main

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type Runnable struct {
	path        string
	isRunning   atomic.Bool
	initialized atomic.Bool
	cancel      context.CancelFunc
	scriptSet   RevolverScriptConfig
}

func (r *Runnable) IsRunning() bool {
	return r.isRunning.Load()
}

func (r *Runnable) IsInitialized() bool {
	return r.initialized.Load()
}

func NewRunnable(path string, script RevolverScriptConfig) *Runnable {
	return &Runnable{
		path:      path,
		scriptSet: script,
	}
}

func (r *Runnable) Start(ctx context.Context, env []string, f func(context.Context, []string, string, RevolverScriptConfig) error) bool {
	if r.IsRunning() {
		return false
	}

	if !r.isRunning.CompareAndSwap(false, true) {
		return false
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	go func() {
		defer r.isRunning.Store(false)
		r.initialized.Store(true)
		r.isRunning.Store(true)
		if err := f(ctx, env, r.path, r.scriptSet); err != nil {
			type ExitError interface {
				ExitCode() int
				Exited() bool
				Error() string
			}
			if errors.As(err, new(ExitError)) {
				log.Debug().Err(err).Msg("failed to run command")
				return
			}
			if errors.Is(err, context.Canceled) {
				log.Debug().Err(err).Msg("cancelled runnable")
				return
			}
			log.Error().Err(err).Msg("stopped runnable")
		}
	}()

	return true
}

func (r *Runnable) Stop() bool {
	if r == nil {
		return true
	}

	if !(r.IsRunning() && r.IsInitialized()) {
		return false
	}

	r.cancel()

	return true
}

func (r *Runnable) WaitForStop() {
	for r.IsRunning() {
		time.Sleep(time.Millisecond * 100)
	}
}
