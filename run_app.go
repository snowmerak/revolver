package main

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type Runnable struct {
	path      string
	isRunning atomic.Bool
	cancel    context.CancelFunc
	scriptSet RevolverScriptConfig
	portSet   []RevolverPortConfig
}

func (r *Runnable) IsRunning() bool {
	return r.isRunning.Load()
}

func NewRunnable(path string, script RevolverScriptConfig, port []RevolverPortConfig) *Runnable {
	return &Runnable{
		path:      path,
		scriptSet: script,
		portSet:   port,
	}
}

func (r *Runnable) Start(ctx context.Context, f func(context.Context, []string, string, RevolverScriptConfig) error) bool {
	if r.IsRunning() {
		return false
	}

	if !r.isRunning.CompareAndSwap(false, true) {
		return false
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.isRunning.Store(true)

	env := make([]string, 0, len(r.portSet))
	for _, port := range r.portSet {
		freePort, err := GetFreeTcpPort()
		if err != nil {
			log.Error().Err(err).Msg("failed to get free port")
			return false
		}

		env = append(env, port.Env+"="+strconv.FormatInt(int64(freePort), 10))
	}

	go func() {
		defer r.isRunning.Store(false)
		if err := f(ctx, env, r.path, r.scriptSet); err != nil {
			log.Error().Err(err).Msg("stopped runnable")
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
