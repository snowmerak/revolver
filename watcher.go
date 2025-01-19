package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type WatcherEvent struct {
	Id      string
	Handler func(*fsnotify.Event)
}

type WatcherError struct {
	Id      string
	Handler func(error)
}

type WatcherConfig struct {
	Path                string
	ExtensionFilter     []string
	ExtensionFilterFunc func(string) bool
}

type Watcher struct {
	watcher           *fsnotify.Watcher
	eventHandlers     []*WatcherEvent
	eventHandlersLock sync.RWMutex
	errHandlers       []*WatcherError
	errHandlersLock   sync.RWMutex
	config            *WatcherConfig
}

func WithPath(path string) func(*WatcherConfig) {
	return func(wc *WatcherConfig) {
		wc.Path = path
	}
}

// WithExtensionFilter allows you to provide a list of file extensions
// that will be used to filter the files.
func WithExtensionFilter(extensions ...string) func(*WatcherConfig) {
	return func(wc *WatcherConfig) {
		wc.ExtensionFilter = extensions
	}
}

// WithExtensionFilterFunc allows you to provide a custom filter function
// that will be used to filter the file extensions.
// The function should return true if the extension is allowed, false otherwise.
func WithExtensionFilterFunc(filter func(string) bool) func(*WatcherConfig) {
	return func(wc *WatcherConfig) {
		wc.ExtensionFilterFunc = filter
	}
}

func NewWatcher(ctx context.Context, opt ...func(*WatcherConfig)) (*Watcher, error) {
	cfg := &WatcherConfig{}
	for _, o := range opt {
		o(cfg)
	}

	wc, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	context.AfterFunc(ctx, func() {
		wc.Close()
	})

	w := &Watcher{
		watcher: wc,
		config:  cfg,
	}

	return w, nil
}

func (w *Watcher) AddEventHandler(id string, handler func(*fsnotify.Event)) {
	w.eventHandlersLock.Lock()
	defer w.eventHandlersLock.Unlock()
	w.eventHandlers = append(w.eventHandlers, &WatcherEvent{
		Id:      id,
		Handler: handler,
	})
}

func (w *Watcher) RemoveEventHandler(id string) {
	w.eventHandlersLock.Lock()
	defer w.eventHandlersLock.Unlock()
	for i, handler := range w.eventHandlers {
		if handler.Id == id {
			w.eventHandlers = append(w.eventHandlers[:i], w.eventHandlers[i+1:]...)
			return
		}
	}
}

func (w *Watcher) AddErrorHandler(id string, handler func(error)) {
	w.errHandlersLock.Lock()
	defer w.errHandlersLock.Unlock()
	w.errHandlers = append(w.errHandlers, &WatcherError{
		Id:      id,
		Handler: handler,
	})
}

func (w *Watcher) RemoveErrorHandler(id string) {
	w.errHandlersLock.Lock()
	defer w.errHandlersLock.Unlock()
	for i, handler := range w.errHandlers {
		if handler.Id == id {
			w.errHandlers = append(w.errHandlers[:i], w.errHandlers[i+1:]...)
			return
		}
	}
}

func (w *Watcher) Watch(ctx context.Context) error {
	if err := w.watcher.AddWith(w.config.Path); err != nil {
		return fmt.Errorf("failed to watch path: %w", err)
	}

	done := ctx.Done()
	closedChannelCount := 0

	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					closedChannelCount++
					if closedChannelCount >= 2 {
						return
					}
					continue
				}

				ext := filepath.Ext(event.Name)
				if len(w.config.ExtensionFilter) != 0 || w.config.ExtensionFilterFunc != nil {
					found := false
					for _, filter := range w.config.ExtensionFilter {
						if strings.HasSuffix(ext, filter) {
							found = true
							break
						}
					}

					if !found && w.config.ExtensionFilterFunc != nil {
						if w.config.ExtensionFilterFunc(ext) {
							found = true
						}
					}

					if !found {
						continue
					}
				}

				w.eventHandlersLock.RLock()
				for _, handler := range w.eventHandlers {
					handler.Handler(&event)
				}
				w.eventHandlersLock.RUnlock()
			case err, ok := <-w.watcher.Errors:
				if !ok {
					closedChannelCount++
					if closedChannelCount >= 2 {
						return
					}
					continue
				}

				w.errHandlersLock.RLock()
				for _, handler := range w.errHandlers {
					handler.Handler(err)
				}
				w.errHandlersLock.RUnlock()
			case <-done:
				return
			}
		}
	}()

	return nil
}
