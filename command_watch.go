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

	Init(cfg.LogLevel)

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
			log.Info().Str("port", port.Name).Str("env", port.Env).Int("port", port.Port).Msg("starting reverse proxy")
			if err := rp.Start(ctx); err != nil {
				log.Error().Err(err).Msg("failed to start reverse proxy")
				panic("occurred critical error!!")
			}
		}()
		rpm[port.Name] = rp
	}

	currentRunnable := NewRunnable(cfg.ExecutablePackageFolder, cfg.Scripts)
	processing := atomic.Bool{}
	restartFunc := func(event *fsnotify.Event) {
		ctx, cancel := context.WithCancel(ctx)

		switch event {
		case nil:
			log.Info().Msg("initializing")
		default:
			log.Info().Str("filename", event.Name).Any("op", event.Op).Msg("file changes detected")
		}

		if !processing.CompareAndSwap(false, true) {
			log.Info().Msg("already processing")
			return
		}

		log.Info().Msg("processing changes")

		defer processing.Store(false)

		previousRunnable := currentRunnable

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

		portEnvMap := map[string]string{}
		for _, port := range cfg.Ports {
			portEnvMap[port.Name] = port.Env
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

		env := make([]string, 0, len(portMap))
		for name, port := range portMap {
			env = append(env, portEnvMap[name]+"="+strconv.FormatInt(int64(port), 10))
		}

		newRunnable := NewRunnable(cfg.ExecutablePackageFolder, cfg.Scripts)
		if !newRunnable.Start(ctx, env, RunCommandSet) {
			cancel()
			log.Error().Msg("failed to start new runnable")
			return
		}

		log.Info().Msg("started new runnable")

		currentRunnable = newRunnable

		previousRunnable.Stop()
	}
	restartFunc(nil)
	wc.AddEventHandler("restart", restartFunc)

	if err := wc.Watch(ctx); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	<-ctx.Done()
	log.Info().Msg("shutting down")
	time.Sleep(5 * time.Second)

	return nil
}
