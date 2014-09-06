package command

import (
	"bytes"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

type testEvent struct {
	fileName string
}

func (t *testEvent) FileName() string {
	return t.fileName
}

func TestParseCmdArgs(t *testing.T) {
	Convey("Argument parsing", t, func() {
		ev := &testEvent{"foo"}

		Convey("Argument without a file extension gives the correct command", func() {
			name, args := parseCmdArgs("echo", ev)
			So(name, ShouldEqual, "echo")
			So(args, ShouldBeEmpty)
		})

		Convey("File name can be injected", func() {
			name, args := parseCmdArgs("go test {{fileName}}", ev)
			So(name, ShouldEqual, "go")
			So(args, ShouldResemble, []string{"test", "foo"})
		})

		Convey("A .go file argument will run 'go run' ", func() {
			name, args := parseCmdArgs("test_tasks/golang.go", nil)
			So(name, ShouldEqual, "go")
			So(args, ShouldResemble, []string{"run", "test_tasks/golang.go"})
		})

		Convey("A .js file argument will run 'node'", func() {
			name, args := parseCmdArgs("test_tasks/javascript.js {{fileName}}", ev)
			So(name, ShouldEqual, "node")
			So(args, ShouldResemble, []string{"test_tasks/javascript.js", "foo"})

		})
	})
}

func TestTerminatingCommand(t *testing.T) {
	Convey("Given a command that terminates", t, func() {
		buf := bytes.Buffer{}

		Convey("Sending an Event input runs the command", func() {
			c := NewCommand(&Opts{
				Args:          "echo -n {{fileName}}",
				NoClearScreen: true,
			})
			c.Stdout = &buf

			c.Events <- &testEvent{"foo.go"}
			<-c.Done
			So(buf.String(), ShouldEqual, "foo.go")

			buf.Reset()
			c.Events <- &testEvent{"bar.go"}
			<-c.Done
			So(buf.String(), ShouldEqual, "bar.go")
		})

		Convey("Errors can be captured", func() {
			c := NewCommand(&Opts{
				Args:          "test_commands/error.go",
				NoClearScreen: true,
			})
			c.Stderr = &buf

			c.Events <- &testEvent{"foo.go"}
			err := <-c.Errors
			So(err.Error(), ShouldContainSubstring, "exit status 1")
			So(buf.String(), ShouldContainSubstring, "found 'EOF'")
		})

		Convey("Timeout can be set", func() {
			c := NewCommand(&Opts{
				Args:          "sleep 1",
				Timeout:       -1,
				NoClearScreen: true,
			})

			c.Events <- &testEvent{"foo.go"}
			<-c.Done
			So(c.DidTimeout, ShouldBeTrue)
		})
	})
}

func TestBlockingCommand(t *testing.T) {
	Convey("Given a command that blocks", t, func() {
		c := NewCommand(&Opts{
			Args:          "test_commands/node.js",
			Blocking:      true,
			NoClearScreen: true,
		})

		Convey("Sending an Event input runs the command", func() {
			c.Events <- &testEvent{"console.log('foo');console.log('fooz');"}
			time.Sleep(time.Millisecond * 80)

			So(string(c.ReadOutPipe()), ShouldEqual, "foo\nfooz\n")
		})

		Convey("Errors can be captured", func() {
			c.Events <- &testEvent{"foo"}
			time.Sleep(time.Millisecond * 80)

			So(string(c.ReadErrPipe()), ShouldContainSubstring, "foo is not defined")
		})

		Convey("A delimeter can be used to help read output", func() {
			c := NewCommand(&Opts{
				Args:          "test_commands/node.js",
				Delim:         "__DONE__",
				NoClearScreen: true,
			})

			buf := bytes.Buffer{}
			c.Stdout = &buf

			c.Events <- &testEvent{"console.log('foo');console.log('fooz__DONE__');"}
			<-c.Done
			So(buf.String(), ShouldEqual, "foo\nfooz\n")

			buf.Reset()
			c.Events <- &testEvent{"console.log('bar');console.log('barz__DONE__');"}
			<-c.Done
			So(buf.String(), ShouldEqual, "bar\nbarz\n")

			buf.Reset()
			c.Stderr = &buf
			c.Events <- &testEvent{"process.stderr.write('errz__DONE__');"}
			<-c.Done
			So(buf.String(), ShouldEqual, "errz")
		})
	})
}

// func TestFoo(t *testing.T) {
// 	opts := &Opts{
// 		Args:          "../gotest/gotest.go",
// 		Blocking:      true,
// 		UseStdout:     true,
// 		NoClearScreen: true,
// 	}

// 	cmd := NewCommand(opts)

// 	cmd.Events <- &event{"../watcher/watcher.go"}
// 	cmd.Events <- &event{"../task/task.go"}

// 	time.Sleep(time.Second * 2)
// }
