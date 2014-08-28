package task

import (
	// "bufio"
	// "errors"
	// "io"
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
	Args      string
	Dir       string
	Timeout   time.Duration
	UsePipeIO bool
}

func NewSimpleTask(opts *Opts) *Task {
	t := &Task{
		Opts:      opts,
		In:        make(chan []byte),
		Done:      make(chan bool),
		ShouldRun: true,
		BeforeRun: func([]byte) {},
		AfterRun:  func() {},
	}

	go t.listen()
	return t
}

type Task struct {
	*Opts
	Cmd       *exec.Cmd
	In        chan []byte
	InPipe    io.WriteCloser
	OutPipe   io.ReadCloser
	ErrPipe   io.ReadCloser
	Timeout   <-chan time.Time
	Done      chan bool
	ShouldRun bool
	BeforeRun func([]byte)
	AfterRun  func()
}

func (p *Task) Kill() error {
	return p.Cmd.Process.Kill()
}

func (t *Task) createCommand(input []byte) {
	t.Args = strings.Replace(t.Args, "{{fileName}}", string(input), -1)

	cmd := Command(t.Args)
	t.Cmd = cmd

	if !t.UsePipeIO {
		t.Cmd.Stdout = os.Stdout
		t.Cmd.Stderr = os.Stderr
	} else {
		out, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal("[error] Problem obtaining StdoutPipe", err)
		}
		t.OutPipe = out

		errPipe, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal("[error] Problem obtaining StderrPipe", err)
		}
		t.ErrPipe = errPipe
	}
}

func (t *Task) listen() {
	for {
		in := <-t.In
		t.createCommand(in)
		t.BeforeRun(in)

		if t.ShouldRun {
			t.Timeout = time.After(time.Second * t.Opts.Timeout)

			go func() {
				t.Cmd.Run() // ? what to do with error
				t.Done <- true
			}()

			t.AfterRun()
		}
	}
}

// func NewTask() *Task {
// 	t := &Task{
// 		// Cmd: cmd,

// 		OnDone:  func() {},
// 		OnError: func(error) {},

// 		Done:   make(chan bool),
// 		Errors: make(chan error),

// 		Timeout: 600,
// 	}

// 	return t
// }

// type Task struct {
// 	Cmd *exec.Cmd

// 	OnDone  func()
// 	OnError func(error)

// 	In      chan string
// 	Out     chan string
// 	Errors  chan error
// 	Done    chan bool
// 	Timeout time.Duration

// 	inPipe io.WriteCloser
// 	outBuf *bufio.Reader
// 	errBuf *bufio.Reader
// }

// // TODO test this
// func (p *Task) RunWithStdIO() {
// 	p.Cmd.Stdin = os.Stdin
// 	p.Cmd.Stdout = os.Stdout
// 	p.Cmd.Stderr = os.Stderr

// 	go func() {
// 		if err := p.Cmd.Run(); err != nil {
// 			p.Errors <- err
// 		} else {
// 			close(p.Done)
// 		}
// 	}()

// 	select {
// 	case <-p.Done:
// 		p.OnDone()
// 	case err := <-p.Errors:
// 		p.OnError(err)
// 	case <-time.After(time.Second * p.Timeout):
// 		log.Println("[info] Long run detected. Halting.")
// 		if err := p.Kill(); err != nil {
// 			log.Fatal("[error]", err)
// 		}
// 	}
// }

// func (p *Task) RunWithPipeIO() error {
// 	in, err := p.Cmd.StdinPipe()
// 	if err != nil {
// 		return err
// 	}

// 	out, err := p.Cmd.StdoutPipe()
// 	if err != nil {
// 		return err
// 	}
// 	outBuf := bufio.NewReader(out)

// 	e, err := p.Cmd.StderrPipe()
// 	if err != nil {
// 		return err
// 	}
// 	errBuf := bufio.NewReader(e)

// 	p.In = make(chan string)
// 	p.Out = make(chan string)
// 	p.inPipe = in
// 	p.outBuf = outBuf
// 	p.errBuf = errBuf

// 	go p.listenIn()
// 	return p.Cmd.Start()
// }

// func (p *Task) listenIn() {
// 	for {
// 		in := <-p.In

// 		_, err := p.inPipe.Write([]byte(in))
// 		logFatal("inPipe write", err)

// 		go p.listenOutBuf()
// 		go p.listenErrBuf()
// 	}
// }

// func (p *Task) listenOutBuf() {
// 	str := p.readFromBuf(p.outBuf)
// 	p.Out <- str
// }

// func (p *Task) listenErrBuf() {
// 	str := p.readFromBuf(p.errBuf)
// 	p.Errors <- errors.New(str)
// }

// func (p *Task) readFromBuf(buf *bufio.Reader) string {
// 	res := []string{}

// 	for {
// 		out, err := buf.ReadString('\n')
// 		if err != nil && err != io.EOF {
// 			logFatal("Output buffer read", err)
// 			return ""
// 		}

// 		remaining := buf.Buffered()

// 		if err == io.EOF && remaining == 0 {
// 			return ""
// 		}

// 		trim := strings.TrimSpace(out)
// 		if trim != "" {
// 			res = append(res, trim)
// 		}

// 		if remaining == 0 && len(res) > 0 {
// 			return strings.Join(res, "\n")
// 		}
// 	}
// }

func logFatal(msg string, err error) {
	if err != nil {
		log.Fatal(msg+" error:", err)
	}
}
