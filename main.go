package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal().Str("guide", "revolver <project-path> <cmd-path> <ext:go is not needed>...").Msg("not enough arguments")
		return
	}

	projectPath := os.Args[1]
	cmdPath := os.Args[2]
	exts := os.Args[3:]

	notContainGo := true
	for _, ext := range exts {
		if strings.HasSuffix(ext, "go") {
			notContainGo = false
		}
	}
	if notContainGo {
		exts = append(exts, ".go")
	}

	log.Info().Str("project_path", projectPath).Str("cmd_path", cmdPath).Strs("exts", exts).Msg("checking arguments")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	rb := NewRunnable(cmdPath)
	if !rb.Start(ctx, RunGoApp) {
		log.Fatal().Msg("failed to start runnable")
		return
	}

	log.Info().Msg("starting watcher")
	w, err := NewWatcher(ctx, WithPath(projectPath), WithExtensionFilter(".go"))
	if err != nil {
		log.Err(err).Msg("failed to create watcher")
		return
	}

	w.AddEventHandler("log", func(event *fsnotify.Event) {
		log.Info().Str("event_data", event.String()).Msg("event received")
	})

	type DebounceState struct {
		checking         atomic.Bool
		debounceDuration time.Duration
	}
	w.AddEventHandler("restart", WrapWatcherHandler(&DebounceState{
		debounceDuration: 5 * time.Second,
	}, func(ds *DebounceState, e *fsnotify.Event) {
		if ds.checking.Load() {
			return
		}

		if !ds.checking.CompareAndSwap(false, true) {
			return
		}

		time.AfterFunc(ds.debounceDuration, func() {
			ds.checking.Store(false)
			log.Info().Msg("restarting app")

			rb.Stop()
			rb.WaitForStop()
			if !rb.Start(ctx, RunGoApp) {
				log.Fatal().Msg("failed to restart runnable")
				return
			}
		})
	}))

	errCount := atomic.Int64{}
	w.AddErrorHandler("log", func(err error) {
		log.Err(err).Msg("error received")
		errCount.Add(1)

		if errCount.Load() > 5 {
			log.Fatal().Msg("too many errors")
			cancel()
		}
	})

	if err := w.Watch(ctx); err != nil {
		log.Err(err).Msg("failed to start watcher")
		return
	}

	<-ctx.Done()
	time.Sleep(1 * time.Second)
}
