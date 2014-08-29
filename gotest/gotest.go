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

var baseDir string
var runSimple *task.SingleTask

func init() {
	wd, err := os.Getwd()
	if err != nil {
		log.Println("[error] Unable to get current working directory to run all tests", err)
		return
	}
	baseDir = wd
}

func NewTask(w *watcher.Watcher, opts *task.Opts) (t *task.SingleTask) {
	t = task.NewSingleTask(opts)

	t.BeforeRun = func(path []byte) {
		w.IgnoreEvents = true

		dir := filepath.Dir(string(path))

		if !hasTestFiles(dir) {
			t.Cmd.Args = []string{"go", "build"}
			fmt.Println("No test files found. Checking for build errors instead.")
		}

		t.Cmd.Dir = dir
	}

	t.OnSuccess = func() {
		if t.Cmd.Args[1] == "build" {
			fmt.Println("Build check complete.")
		}
		runAll(w, t.Cmd.Dir)
	}

	t.OnTimeout = func() {
		killTask(t)
	}

	t.OnDone = func() {
		w.IgnoreEvents = false
	}

	runSimple = createRunSimple(w, opts)
	runAll(w, "")
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

func killTask(t *task.SingleTask) {
	log.Println("Timed out. Halting run.")
	err := t.Kill()
	if err != nil {
		log.Println("[error] Unable to halt test run.", err)
	}
}

func createRunSimple(w *watcher.Watcher, opts *task.Opts) *task.SingleTask {
	t := task.NewSingleTask(opts)

	t.BeforeRun = func(path []byte) {
		w.IgnoreEvents = true
		t.Cmd.Dir = string(path)
	}

	t.OnTimeout = func() {
		killTask(t)
	}

	t.OnDone = func() {
		w.IgnoreEvents = false
	}

	return t
}

func runAll(w *watcher.Watcher, skip string) {
	t := runSimple
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		absSkip, err := filepath.Abs(skip)
		if err != nil {
			log.Println("[error] Unable to get absolute path", err)
		}

		if info.IsDir() && w.IsWatchingDir(path) && hasTestFiles(path) && path != absSkip {
			t.NoClearScreen = true
			t.In <- []byte(path)
			t.NoClearScreen = false
		} else {
			if path != baseDir && info.IsDir() {
				return filepath.SkipDir
			}
		}
		return err
	})
	if err != nil {
		log.Println("[error] Unable to find test directories:", err)
	}
}
