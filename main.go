package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <Command>", os.Args[0])
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case CommandInit:
		if err := CommandInitFunc(args); err != nil {
			log.Error().Err(err).Strs("args", args).Msg("failed to run command")
		}
	}

	//log.Info().Str("project_path", projectPath).Str("cmd_path", cmdPath).Strs("exts", exts).Msg("checking arguments")
	//
	//ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	//defer cancel()
	//
	//rb := NewRunnable(cmdPath, RevolverScriptConfig{})
	//
	//log.Info().Msg("starting watcher")
	//w, err := NewWatcher(ctx, WithPath(projectPath), WithExtensionFilter(".go"))
	//if err != nil {
	//	log.Err(err).Msg("failed to create watcher")
	//	return
	//}
	//
	//w.AddEventHandler("log", func(event *fsnotify.Event) {
	//	log.Info().Str("event_data", event.String()).Msg("event received")
	//})
	//
	//type DebounceState struct {
	//	checking         atomic.Bool
	//	debounceDuration time.Duration
	//}
	//w.AddEventHandler("restart", WrapWatcherHandler(&DebounceState{
	//	debounceDuration: 2 * time.Second,
	//}, func(ds *DebounceState, e *fsnotify.Event) {
	//	if ds.checking.Load() {
	//		return
	//	}
	//
	//	if !ds.checking.CompareAndSwap(false, true) {
	//		return
	//	}
	//
	//	time.AfterFunc(ds.debounceDuration, func() {
	//		ds.checking.Store(false)
	//		log.Info().Msg("restarting app")
	//
	//		rb.Stop()
	//		rb.WaitForStop()
	//	})
	//}))
	//
	//if err := w.Watch(ctx); err != nil {
	//	log.Err(err).Msg("failed to start watcher")
	//	return
	//}
	//
	//<-ctx.Done()
	//time.Sleep(1 * time.Second)
}
