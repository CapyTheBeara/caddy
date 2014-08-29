package main

import (
	"log"

	"github.com/monocle/caddy/gotest"
	"github.com/monocle/caddy/task"
	"github.com/monocle/caddy/watcher"
)

func main() {
	log.SetFlags(log.Lshortfile)

	goWatcher := watcher.NewWatcher(&watcher.Config{
		Dir:         ".",
		Ext:         "go",
		ExcludeDirs: []string{".git", "tmp*", "node_module*"},
	})

	jsWatcher := watcher.NewWatcher(&watcher.Config{
		Dir:         ".",
		Ext:         "js",
		ExcludeDirs: []string{".git", "tmp*", "node_module*"},
	})

	goTask := gotest.NewTask(goWatcher, &task.Opts{
		Args:    "go test",
		Timeout: 5,
	})

	jsTask := task.NewSingleTask(&task.Opts{
		Args: "jshint {{fileName}}",
	})

	go goWatcher.Watch(task.NewTasks([]task.Task{goTask}))
	go jsWatcher.Watch(task.NewTasks([]task.Task{jsTask}))

	<-make(chan struct{})
}
