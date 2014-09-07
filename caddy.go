package main

import (
	"io/ioutil"
	"log"

	"github.com/monocle/caddy/config"
)

func main() {
	log.SetFlags(log.Lshortfile)

	cfg, err := ioutil.ReadFile("caddy.json")
	if err != nil {
		log.Fatal("[error] Unable to redy caddy.json file")
	}
	config.ParseConfig(cfg)
	<-make(chan struct{})
}
