package watcher

import (
	"bufio"
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var commandMap = map[string][]string{
	".go": []string{"go", "run"},
	".js": []string{"node"},
}

type Event interface {
	FileName() string
}

func parseCmdArgs(argStr string, ev Event) (string, []string) {
	a := argStr
	if ev != nil {
		a = strings.Replace(argStr, "{{fileName}}", ev.FileName(), -1)
	}
	args := strings.Split(a, " ")

	ext := filepath.Ext(args[0])
	if ext != "" {
		args = append(commandMap[ext], args...)
	}
	return args[0], args[1:]
}

type Opts struct {
	Args     string
	Timeout  time.Duration
	UsePipes bool
	Delim    []byte
}

type Cmd struct {
	*Opts
	*exec.Cmd
	Errors     chan error
	Events     chan Event
	Done       chan struct{}
	Timeout    chan time.Duration
	DidTimeout bool

	inPipe  io.WriteCloser
	OutPipe io.ReadCloser
	ErrPipe io.ReadCloser
}

func (c *Cmd) ReadOutPipe() []byte {
	return c.readPipe(c.OutPipe)
}

func (c *Cmd) ReadErrPipe() []byte {
	return c.readPipe(c.ErrPipe)
}

func (c *Cmd) readPipe(r io.ReadCloser) []byte {
	res := []byte{}
	bufr := bufio.NewReader(r)

	for {
		p, err := readBufOnce(bufr)
		if err != nil {
			println(err.Error())
			c.Errors <- err
			break
		}
		res = append(res, p...)

		if len(c.Delim) == 0 {
			break
		}

		if bytes.Contains(res, c.Delim) {
			res = bytes.Replace(res, c.Delim, []byte{}, 1)
			break
		}
	}
	return res
}

func readBufOnce(bufr *bufio.Reader) ([]byte, error) {
	_, err := bufr.Peek(1)
	if err != nil {
		return []byte{}, err
	}
	buf := make([]byte, bufr.Buffered())
	bufr.Read(buf)
	return buf, nil
}

func (c *Cmd) listen() {
	for {
		ev := <-c.Events
		name, args := parseCmdArgs(c.Opts.Args, ev)
		cmd := exec.Command(name, args...)
		cmd.Stdout = c.Stdout
		cmd.Stderr = c.Stderr
		c.Cmd = cmd

		done := make(chan struct{})

		go func() {
			err := c.Run()
			if err != nil {
				c.Errors <- err
			}
			done <- struct{}{}
		}()

		select {
		case <-done:
		case <-time.After(time.Second * c.Opts.Timeout):
			c.DidTimeout = true
		}
		c.Done <- struct{}{}
	}
}

func (c *Cmd) listenPipes() {
	for {
		ev := <-c.Events

		_, err := c.inPipe.Write([]byte(ev.FileName()))
		if err != nil {
			c.Errors <- err
		}

		if len(c.Delim) > 0 {
			go func() {
				res := c.ReadOutPipe()
				c.Stdout.Write(res)
				c.Done <- struct{}{}
			}()

			go func() {
				res := c.ReadErrPipe()
				c.Stderr.Write(res)
				c.Done <- struct{}{}
			}()
		}
	}
}

func (c *Cmd) setupPipes() error {
	in, err := c.Cmd.StdinPipe()
	if err != nil {
		return err
	}
	c.inPipe = in

	out, err := c.Cmd.StdoutPipe()
	if err != nil {
		return err
	}
	c.OutPipe = out

	errPipe, err := c.Cmd.StderrPipe()
	if err != nil {
		return err
	}
	c.ErrPipe = errPipe
	return nil
}

func NewCommand(opts *Opts) *Cmd {
	if opts.Timeout == 0 {
		opts.Timeout = 600
	}
	c := Cmd{
		Cmd:     &exec.Cmd{},
		Opts:    opts,
		Events:  make(chan Event),
		Errors:  make(chan error),
		Done:    make(chan struct{}),
		Timeout: make(chan time.Duration),
	}

	if opts.UsePipes {
		name, args := parseCmdArgs(c.Opts.Args, nil)
		c.Cmd = exec.Command(name, args...)
		err := c.setupPipes()
		if err != nil {
			c.Errors <- err
		} else {
			c.Start()
			go c.listenPipes()
		}
	} else {
		go c.listen()
	}
	return &c
}
