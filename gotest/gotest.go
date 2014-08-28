package gotest

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/monocle/caddy/task"
	"github.com/monocle/caddy/watcher"
)

func NewTask(w *watcher.Watcher, opts *task.Opts) (t *task.Task) {
	t = task.NewSimpleTask(opts)

	t.BeforeRun = func(path []byte) {
		w.IgnoreEvents = true

		fmt.Print("\033[2J\033[0;0H")

		dir := filepath.Dir(string(path))

		if !hasTestFiles(dir) {
			t.Cmd.Args = []string{"go", "build"}
			fmt.Println("No test files found. Checking for build errors instead.")
		}

		t.Cmd.Dir = dir
	}

	t.AfterRun = func() {
		select {
		case <-t.Done:
			if t.Cmd.Args[1] == "build" {
				fmt.Println("Build successful.")
			}
			// dirs := findTestDirs(w)

		case <-t.Timeout:
			log.Println("Timed out. Halting run.")
			t.Kill()
		}
		w.IgnoreEvents = false
	}

	return t
}

func hasTestFiles(dir string) bool {
	yes := false
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if yes {
			return err
		}

		if path != dir && info.IsDir() {
			return filepath.SkipDir
		}

		if strings.HasSuffix(path, "_test.go") {
			yes = true
		}
		return err
	})
	if err != nil {
		log.Println("[error] finding test files:", err)
	}
	return yes
}

func findTestDirs(w *watcher.Watcher) []string {
	dirs := []string{}
	wd, err := os.Getwd()
	if err != nil {
		log.Println("[error] Unable to get current working directory", err)
		return dirs
	}

	err = filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && w.IsWatchingDir(path) && hasTestFiles(path) {
			dirs = append(dirs, path)
		}
		return err
	})
	if err != nil {
		log.Println("[error] Unable to find test directories:", err)
	}
	return dirs
}
