package main

import (
	"log"

	"github.com/monocle/caddy/config"
)

const jsonConf = `
	{
		"commands": [
			{
				"args": "gotest/gotest.go",
				"blocking": true
			},
			{
				"args": "jshint {{fileName}}",
				"ignoreErrors": true
			}
		],
		"watchers": [
			{
				"dir": ".",
				"ext": "go",
				"excludeDirs": [".git", "tmp*", "node_module*"],
				"commands": ["gotest/gotest.go"]
			},
			{
				"dir": ".",
				"ext": "js",
				"excludeDirs": [".git", "tmp*", "node_module*"],
				"commands": ["jshint {{fileName}}"]
			}
		]
	}
`

func main() {
	log.SetFlags(log.Lshortfile)
	config.ParseConfig([]byte(jsonConf))
	<-make(chan struct{})
}
