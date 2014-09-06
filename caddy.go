package main

import (
	"log"

	"github.com/monocle/caddy/command"
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

	goCommand := command.NewCommand(&command.Opts{
		Args:      "gotest/gotest.go",
		Blocking:  true,
		UseStdout: true,
	})

	jsCommand := command.NewCommand(&command.Opts{
		Args:      "jshint {{fileName}}",
		UseStdout: true,
	})

	go func() {
		for {
			select {
			case err := <-goCommand.Errors:
				log.Println("[error] go test", err)
				// if watcher.SupressCommandErrors
				// case err := <-jsCommand.Errors:
				// log.Println("[error] jshint ", err)
			}
		}
	}()

	go goWatcher.AddCommands(command.NewCommands([]*command.Cmd{goCommand}))
	go jsWatcher.AddCommands(command.NewCommands([]*command.Cmd{jsCommand}))

	<-make(chan struct{})
}
