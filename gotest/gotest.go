package gotest

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/monocle/caddy/watcher"
)

var timeout = time.Second * 5

func Run() {
	w := watcher.NewWatcher(&watcher.Config{
		Dir:         ".",
		IgnoreModes: []string{"chmod"},
	})

	for {
		e := <-w.Events
		w.IgnoreEvents = true

		fmt.Print("\033[2J\033[0;0H")

		cmd := exec.Command("go", "test")
		cmd.Dir = filepath.Dir(e.Name)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		done := make(chan bool)

		go func() {
			err := cmd.Start()
			if err != nil {
				log.Println("[error]", err)
			} else {
				cmd.Wait()
				close(done)
			}
		}()

		select {
		case <-done:
		case <-time.After(timeout):
			log.Println("[info] Unblocking long run")
			if err := cmd.Process.Kill(); err != nil {
				log.Fatal("[error]", err)
			}
		}

		w.IgnoreEvents = false
	}
}
