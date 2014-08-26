package plugin

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var commandMap = map[string][]string{
	".go": []string{"go", "run"},
	".js": []string{"node"},
	// ".rb": []string{"ruby"},
	// ".py": []string{"python"},
}

func NewPlugin(args []string) *Plugin {
	baseArgs := commandMap[filepath.Ext(args[0])]
	rest := append(baseArgs[1:], args...)
	cmd := exec.Command(baseArgs[0], rest...)
	p := &Plugin{
		Cmd: cmd,
	}

	return p
}

type Plugin struct {
	*exec.Cmd
	In     chan string
	Out    chan string
	Errors chan error
	inPipe io.WriteCloser
	outBuf *bufio.Reader
	errBuf *bufio.Reader
}

func (p *Plugin) RunWithStdIO() error {
	p.Stdin = os.Stdin
	p.Stdout = os.Stdout
	p.Stderr = os.Stderr

	return p.Run()
}

func (p *Plugin) RunWithPipeIO() error {
	in, err := p.StdinPipe()
	if err != nil {
		return err
	}

	out, err := p.StdoutPipe()
	if err != nil {
		return err
	}
	outBuf := bufio.NewReader(out)

	e, err := p.StderrPipe()
	if err != nil {
		return err
	}
	errBuf := bufio.NewReader(e)

	p.In = make(chan string)
	p.Out = make(chan string)
	p.Errors = make(chan error)
	p.inPipe = in
	p.outBuf = outBuf
	p.errBuf = errBuf

	go p.listenIn()
	return p.Start()
}

func (p *Plugin) listenIn() {
	for {
		in := <-p.In

		_, err := p.inPipe.Write([]byte(in))
		logFatal("inPipe write", err)

		go p.listenOutBuf()
		go p.listenErrBuf()
	}
}

func (p *Plugin) listenOutBuf() {
	str := p.readFromBuf(p.outBuf)
	p.Out <- str
}

func (p *Plugin) listenErrBuf() {
	str := p.readFromBuf(p.errBuf)
	p.Errors <- errors.New(str)
}

func (p *Plugin) readFromBuf(buf *bufio.Reader) string {
	res := []string{}

	for {
		out, err := buf.ReadString('\n')
		if err != nil && err != io.EOF {
			logFatal("Output buffer read", err)
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

func logFatal(msg string, err error) {
	if err != nil {
		log.Fatal(msg+" error:", err)
	}
}
