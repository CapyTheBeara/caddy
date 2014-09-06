package command

type Commands struct {
	Content []*Cmd
}

func (c *Commands) Run(fileName string) {
	for _, c := range c.Content {
		c.Events <- &event{fileName}
	}
}

func NewCommands(cs []*Cmd) *Commands {
	return &Commands{cs}
}
