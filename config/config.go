package config

import (
	"encoding/json"
	"log"

	"github.com/monocle/caddy/command"
	"github.com/monocle/caddy/watcher"
)

type Config struct {
	Commands []*command.Opts
	Watchers []*watcher.Config
	cmds     []*command.Cmd
}

func (c *Config) GetCmd(argStr string) *command.Cmd {
	for _, cmd := range c.cmds {
		if argStr == cmd.Opts.Args {
			return cmd
		}
	}
	log.Fatal("Command not found:", argStr)
	return nil
}

func ParseConfig(cfg []byte) *Config {
	config := Config{}

	err := json.Unmarshal(cfg, &config)
	if err != nil {
		log.Fatalln(err)
	}

	cmds := []*command.Cmd{}
	for _, opts := range config.Commands {
		cmds = append(cmds, command.NewCommand(opts))
	}

	config.cmds = cmds

	for _, wc := range config.Watchers {
		w := watcher.NewWatcher(wc)

		cmds := []*command.Cmd{}
		for _, argStr := range wc.Commands {
			cmds = append(cmds, config.GetCmd(argStr))
		}

		go w.AddCommands(command.NewCommands(cmds))
	}

	return &config
}
