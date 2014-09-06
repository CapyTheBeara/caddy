package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/monocle/caddy/task"
)

func main() {
	onError := func(err error) {
		os.Stderr.WriteString("[error] " + err.Error() + "\n")
	}

	onData := func(path string) {
		cmd := exec.Command("go", "test")
		cmd.Dir = filepath.Dir(path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			onError(err)
		}
	}

	task.StartTask(onData, onError)
}
