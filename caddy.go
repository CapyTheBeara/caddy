package main

import (
	"log"

	"github.com/monocle/caddy/gotest"
	"github.com/monocle/caddy/task"
	"github.com/monocle/caddy/watcher"
)

func main() {
	log.SetFlags(log.Lshortfile)

	println("Running...")

	goWatcher := watcher.NewWatcher(&watcher.Config{
		Dir:         ".",
		Ext:         "go",
		ExcludeDirs: []string{".git", "tmp*", "node_module*"},
	})

	gotestTask := gotest.NewTask(goWatcher, &task.Opts{
		Args:    "go test",
		Timeout: 5,
	})

	jsWatcher := watcher.NewWatcher(&watcher.Config{
		Dir:         ".",
		Ext:         "js",
		ExcludeDirs: []string{".git", "tmp*", "node_module*"},
	})

	jshintTask := task.NewSimpleTask(&task.Opts{
		Args: "jshint {{fileName}}",
	})

	for {
		select {
		case e := <-goWatcher.Events:
			gotestTask.In <- []byte(e.Name)
		case e := <-jsWatcher.Events:
			jshintTask.In <- []byte(e.Name)
		}
	}
}
