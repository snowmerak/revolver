package main

import "github.com/fsnotify/fsnotify"

func WrapWatcherHandler[T any](initState T, callback func(T, *fsnotify.Event)) func(*fsnotify.Event) {
	return func(event *fsnotify.Event) {
		callback(initState, event)
	}
}
