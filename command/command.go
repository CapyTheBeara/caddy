package command

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var commandMap = map[string][]string{
	".go": []string{"go", "run"},
	".js": []string{"node"},
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
	Args          string
	NoClearScreen bool
	Timeout       time.Duration

	Delim     string
	Blocking  bool
	UseStdout bool
}

func NewCommand(opts *Opts) (c *Cmd) {
	if opts.Timeout == 0 {
		opts.Timeout = 600
	}

	if opts.Delim != "" {
		opts.Blocking = true
	}

	c = &Cmd{
		Cmd:    &exec.Cmd{},
		Opts:   opts,
		Events: make(chan Event),
		Done:   make(chan struct{}),
	}

	if opts.Blocking {
		c.listenBlocking()
		return
	}

	c.listen()
	return
}

type Cmd struct {
	*Opts
	*exec.Cmd
	Errors     chan error
	Events     chan Event
	Done       chan struct{}
	Timeout    chan time.Duration
	DidTimeout bool

	OutPipe io.ReadCloser
	ErrPipe io.ReadCloser
}

func (c *Cmd) listen() {
	c.Errors = make(chan error)
	c.Timeout = make(chan time.Duration)

	go func() {
		for {
			ev := <-c.Events
			if !c.NoClearScreen {
				fmt.Print("\033[2J\033[0;0H")
			}

			name, args := parseCmdArgs(c.Opts.Args, ev)
			cmd := exec.Command(name, args...)
			cmd.Stdout = c.Stdout
			cmd.Stderr = c.Stderr

			c.Cmd = cmd
			if c.UseStdout {
				c.useOsStdout()
			}

			done := make(chan struct{})

			go func() {
				c.checkError(c.Run())
				done <- struct{}{}
			}()

			select {
			case <-done:
			case <-time.After(time.Second * c.Opts.Timeout):
				c.DidTimeout = true
			}

			go func() {
				c.Done <- struct{}{}
			}()
		}
	}()
}

func (c *Cmd) useOsStdout() {
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
}

func (c *Cmd) ReadOutPipe() []byte {
	return c.readPipe(c.OutPipe)
}

func (c *Cmd) ReadErrPipe() []byte {
	return c.readPipe(c.ErrPipe)
}

func (c *Cmd) readPipe(r io.ReadCloser) []byte {
	bufr := bufio.NewReader(r)
	p, err := readBufOnce(bufr)
	c.checkError(err)
	return p
}

func (c *Cmd) readPipeDelim(r io.ReadCloser) []byte {
	res := []byte{}
	bufr := bufio.NewReader(r)

	for {
		p, err := readBufOnce(bufr)
		c.checkError(err)

		res = append(res, p...)

		delim := []byte(c.Delim)
		if bytes.Contains(res, delim) {
			res = bytes.Replace(res, delim, []byte{}, 1)
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

func (c *Cmd) listenBlocking() {
	name, args := parseCmdArgs(c.Opts.Args, nil)
	c.Cmd = exec.Command(name, args...)

	inPipe, err := c.Cmd.StdinPipe()
	c.checkError(err)

	if c.UseStdout {
		c.useOsStdout()
	} else {
		c.checkError(c.useStdoutPipe())
	}

	c.Start()

	go func() {
		for {
			ev := <-c.Events
			if !c.NoClearScreen {
				fmt.Print("\033[2J\033[0;0H")
			}

			_, err := inPipe.Write([]byte(ev.FileName() + "\n"))
			c.checkError(err)

			if c.Delim != "" {
				go func() {
					res := c.readPipeDelim(c.OutPipe)
					c.Stdout.Write(res)
					c.Done <- struct{}{}
				}()

				go func() {
					res := c.readPipeDelim(c.ErrPipe)
					c.Stderr.Write(res)
					c.Done <- struct{}{}
				}()
			}
		}
	}()
}

func (c *Cmd) useStdoutPipe() error {
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

func (c *Cmd) checkError(err error) bool {
	if err != nil {
		go func() {
			c.Errors <- err
		}()
		return true
	}
	return false
}
