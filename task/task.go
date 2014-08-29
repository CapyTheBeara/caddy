package task

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
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

func Command(argStr string) *exec.Cmd {
	var cmd *exec.Cmd
	args := strings.Split(argStr, " ")

	ext := filepath.Ext(args[0])
	if ext == "" {
		if len(args) == 0 {
			cmd = exec.Command(args[0])
		} else {
			cmd = exec.Command(args[0], args[1:]...)
		}
	} else {
		baseArgs := commandMap[ext]
		rest := append(baseArgs[1:], args...)
		cmd = exec.Command(baseArgs[0], rest...)
	}
	return cmd
}

type Opts struct {
	Args          string
	Dir           string
	Timeout       time.Duration
	UsePipeIO     bool
	NoClearScreen bool
}

type Task interface {
	Input() chan []byte
	Kill() error
	Done() chan struct{}
}

type common struct {
	*Opts
	Cmd     *exec.Cmd
	In      chan []byte
	OutPipe io.ReadCloser
	ErrPipe io.ReadCloser
}

func (c *common) Input() chan []byte {
	return c.In
}

func (p *common) Kill() error {
	return p.Cmd.Process.Kill()
}

func (t *common) addStdOutErr() {
	t.Cmd.Stdout = os.Stdout
	t.Cmd.Stderr = os.Stderr
}

func (t *common) addPipeOutErr() {
	out, err := t.Cmd.StdoutPipe()
	if err != nil {
		log.Fatal("[error] Problem obtaining StdoutPipe", err)
	}
	t.OutPipe = out
	errPipe, err := t.Cmd.StderrPipe()
	if err != nil {
		log.Fatal("[error] Problem obtaining StderrPipe", err)
	}
	t.ErrPipe = errPipe
}

func NewSingleTask(opts *Opts) *SingleTask {
	if opts.Timeout == 0 {
		opts.Timeout = 600
	}
	t := &SingleTask{
		common: common{
			Opts: opts,
			In:   make(chan []byte),
		},
		Success:   make(chan struct{}),
		Errors:    make(chan error),
		ShouldRun: true,
		BeforeRun: func([]byte) {},
		OnSuccess: func() {},
		OnError:   func(error) {},
		OnTimeout: func() {},
		OnDone:    func() {},
	}

	go t.runCommand()
	return t
}

type SingleTask struct {
	common
	Success   chan struct{}
	Errors    chan error
	done      chan struct{}
	ShouldRun bool
	BeforeRun func([]byte)
	OnSuccess func()
	OnError   func(error)
	OnTimeout func()
	OnDone    func()
}

func (t *SingleTask) Done() chan struct{} {
	return t.done
}

func (t *SingleTask) createCommand(input []byte) {
	t.Args = strings.Replace(t.Args, "{{fileName}}", string(input), -1)
	t.Cmd = Command(t.Args)

	if !t.UsePipeIO {
		t.addStdOutErr()
	} else {
		t.addPipeOutErr()
	}
}

func (t *SingleTask) runCommand() {
	for {
		in := <-t.In
		t.done = make(chan struct{})

		if t.ShouldRun {
			if !t.NoClearScreen {
				fmt.Print("\033[2J\033[0;0H")
			}

			t.createCommand(in)
			t.BeforeRun(in)

			go func() {
				// log.Println("1", t.Cmd.Args)
				err := t.Cmd.Run()
				// log.Println("2", t.Cmd.Args)
				if err != nil {
					t.Errors <- err
					return
				}
				t.Success <- struct{}{}
			}()

			select {
			case <-t.Success:
				t.OnSuccess()
			case err := <-t.Errors:
				t.OnError(err)
			case <-time.After(time.Second * t.Opts.Timeout):
				t.OnTimeout()
			}
		}
		t.OnDone()
		close(t.done)
	}
}

func NewMultiTask(opts *Opts) *MultiTask {
	cmd := Command(opts.Args)

	t := &MultiTask{
		common: common{
			Cmd:  cmd,
			Opts: opts,
			In:   make(chan []byte),
		},

		Out:    make(chan []byte),
		Errors: make(chan error),
	}

	t.addPipeIn()
	t.addPipeOutErr()
	go t.listen()

	cmd.Start()
	return t
}

type MultiTask struct {
	common
	inPipe io.WriteCloser
	Out    chan []byte
	Errors chan error
}

func (t *MultiTask) addPipeIn() {
	in, err := t.Cmd.StdinPipe()
	if err != nil {
		log.Fatal("[error] Problem obtaining StdinPipe", err)
	}
	t.inPipe = in
}

func (t *MultiTask) listen() {
	res := make(chan []byte)
	errc := make(chan []byte)

	for {
		in := <-t.In

		_, err := t.inPipe.Write(in)
		if err != nil {
			log.Println("[error] problem writing to StdinPipe", err)
			continue
		}

		quit := make(chan struct{})
		go readPipe(t.OutPipe, res, quit)
		go readPipe(t.ErrPipe, errc, quit)

		select {
		case p := <-res:
			t.Out <- p
		case p := <-errc:
			t.Errors <- errors.New(string(p))
		}
		close(quit)
	}
}

func readPipe(pipe io.ReadCloser, res chan []byte, quit chan struct{}) {
	select {
	case <-quit:
	default:
		res <- []byte(readFromBuf(pipe))
	}
}

func readFromBuf(rd io.Reader) string {
	buf := bufio.NewReader(rd)
	res := []string{}

	for {
		out, err := buf.ReadString('\n')
		if err != nil && err != io.EOF {
			log.Fatal("Output buffer read", err)
			return ""
		}

		remaining := buf.Buffered()

		if err == io.EOF && remaining == 0 {
			return ""
		}

		trim := strings.TrimSpace(out)
		if trim != "" {
			res = append(res, trim)
		}

		if remaining == 0 && len(res) > 0 {
			return strings.Join(res, "\n")
		}
	}
}

func NewTasks(ts []Task) *Tasks {
	return &Tasks{ts}
}

type Tasks struct {
	content []Task
}

func (t *Tasks) Notify(msg string) {
	for _, task := range t.content {
		task.Input() <- []byte(msg)
	}
}
