package main

import (
	"log"
	"os"

	"github.com/monocle/caddy/gotest"
	"github.com/monocle/caddy/plugin"
)

func main() {
	log.SetFlags(log.Lshortfile)

	println("Running...")

	if len(os.Args) == 1 {
		gotest.Run()
	} else {
		c := plugin.NewPlugin(os.Args[1:])
		if err := c.RunWithStdIO(); err != nil {
			log.Fatal(err)
		}

	}
}
