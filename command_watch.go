package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const CommandWatch = "watch"

var CommandWatchNotFoundFilenameError = fmt.Errorf("missing filename to watch")

func CommandWatchFunc(args []string) error {
	if len(args) < 1 {
		fmt.Printf("Usage: %s %s <filename>\n", os.Args[0], CommandWatch)
		return CommandWatchNotFoundFilenameError
	}

	filename := args[0]

	fmt.Printf("Watching file: %s\n", filename)

	cfg := RevolverConfig{}
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return fmt.Errorf("failed to decode yaml: %w", err)
	}

	log.Info().Str("filename", filename).Any("config", cfg).Msg("watching with config")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	wc, err := NewWatcher(ctx, WithPath(cfg.ProjectRootFolder), WithExtensionFilter(cfg.ObservingExts...))
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	rpm := map[string]*TcpReverseProxy{}
	for _, port := range cfg.Ports {
		rp := NewTcpReverseProxy("0.0.0.0:" + strconv.FormatInt(int64(port.Port), 10))
		go func() {
			if err := rp.Start(ctx); err != nil {
				log.Error().Err(err).Msg("failed to start reverse proxy")
				panic("occurred critical error!!")
			}
		}()
		rpm[port.Name] = rp
	}

	wc.AddEventHandler("log", func(event *fsnotify.Event) {
		log.Info().Str("event_data", event.String()).Msg("event received")
	})

	currentRunnable := NewRunnable(cfg.ExecutablePackageFolder, cfg.Scripts)
	processing := atomic.Bool{}
	wc.AddEventHandler("restart", func(event *fsnotify.Event) {
		if !processing.CompareAndSwap(false, true) {
			return
		}

		defer processing.Store(false)

		previousRunnable := currentRunnable

		ctx, cancel := context.WithCancel(ctx)

		id, err := NewSession()
		if err != nil {
			cancel()
			log.Error().Err(err).Msg("failed to create new session")
			return
		}

		portMap, err := GetFreeTcpPortEnv(cfg.Ports)
		if err != nil {
			cancel()
			log.Error().Err(err).Msg("failed to get free tcp port")
			return
		}

		for name, rp := range rpm {
			if err := rp.RenewDestination(id, "0.0.0.0:"+strconv.FormatInt(int64(portMap[name]), 10), func() {
				cancel()
			}); err != nil {
				cancel()
				log.Error().Err(err).Msg("failed to renew destination")
				return
			}
		}

		newRunnable := NewRunnable(cfg.ExecutablePackageFolder, cfg.Scripts)
		if !newRunnable.Start(ctx, os.Environ(), RunCommandSet) {
			cancel()
			log.Error().Msg("failed to start new runnable")
			return
		}

		currentRunnable = newRunnable

		previousRunnable.Stop()
	})

	if err := wc.Watch(ctx); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	<-ctx.Done()
	log.Info().Msg("shutting down")
	time.Sleep(5 * time.Second)

	return nil
}
