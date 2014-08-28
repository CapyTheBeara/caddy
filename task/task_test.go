package task

import (
	"bytes"
	"io"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCommand(t *testing.T) {
	Convey("Argument parsing", t, func() {
		Convey("Argument without a file extension gives the correct command", func() {
			cmd := Command("go test")
			So(cmd.Args, ShouldResemble, []string{"go", "test"})
		})

		Convey("A .go file argument will run 'go run' ", func() {
			cmd := Command("test_tasks/golang.go")
			So(cmd.Args, ShouldResemble, []string{"go", "run", "test_tasks/golang.go"})
		})

		Convey("A .js file argument will run 'node'", func() {
			cmd := Command("test_tasks/javascript.js")
			So(cmd.Args, ShouldResemble, []string{"node", "test_tasks/javascript.js"})
		})
	})
}

func TestSimpleTask(t *testing.T) {
	Convey("Given a simple task", t, func() {
		Convey("It can specify that the file name be sent as a command arg", func() {
			t := NewSimpleTask(&Opts{
				Args:      "echo {{fileName}}",
				UsePipeIO: true,
			})

			t.In <- []byte("hello")

			var buf bytes.Buffer
			io.Copy(&buf, t.OutPipe)

			So(buf.String(), ShouldEqual, "hello\n")
		})

		Convey("It can detect errors", func() {
			t := NewSimpleTask(&Opts{
				Args:      "test_tasks/error.go",
				UsePipeIO: true,
			})
			t.In <- []byte("hello")

			var buf bytes.Buffer
			io.Copy(&buf, t.ErrPipe)

			So(buf.String(), ShouldContainSubstring, "found 'EOF'")
		})

		Convey("It can time out", func() {
			t := NewSimpleTask(&Opts{
				Args:      "test_tasks/node.js",
				UsePipeIO: true,
				Timeout:   0,
			})

			t.In <- []byte("hello")
			<-t.Timeout
			So("Pass - Timed out", ShouldNotBeBlank)
		})

		Convey("It has a BeforeRun callback", func() {
			t := NewSimpleTask(&Opts{
				Args:      "echo",
				UsePipeIO: true,
			})

			t.BeforeRun = func(b []byte) {
				t.Cmd.Args = append(t.Cmd.Args, "fooz")
			}

			t.In <- []byte("")

			var buf bytes.Buffer
			io.Copy(&buf, t.OutPipe)

			So(buf.String(), ShouldEqual, "fooz\n")
		})

		Convey("It has an AfterRun callback", func() {
			t := NewSimpleTask(&Opts{
				Args:      "test_tasks/node.js",
				UsePipeIO: true,
				Timeout:   0,
			})

			pass := false
			t.AfterRun = func() {
				<-t.Timeout
				So("Pass - Timed out", ShouldNotBeBlank)
				pass = true
			}

			t.In <- []byte("hello")

			<-t.Done
			if !pass {
				So("Fail - should not complete", ShouldBeEmpty)
			}
		})

		Convey("It can prevent a run", func() {
			t := NewSimpleTask(&Opts{
				Args:      "echo foo",
				UsePipeIO: true,
			})
			t.ShouldRun = false

			t.In <- []byte{}

			time.Sleep(time.Millisecond * 20)

			select {
			case <-t.Done:
				So("Fail - task should not run", ShouldBeBlank)
			default:
				So("Pass - task was prevented", ShouldNotBeBlank)
			}
		})
	})
}

// func TestNodeTask(t *testing.T) {
// 	Convey("Given a .js file argument", t, func() {
// 		task := NewTask("test_tasks/foo.js")

// 		Convey("It can use a channel for output", func() {
// 			err := task.RunWithPipeIO()
// 			if err != nil {
// 				t.Fatal(err)
// 			}

// 			task.In <- `console.log('foo\nbar');`
// 			res := <-task.Out
// 			So(res, ShouldEqual, "foo\nbar")
// 		})

// 		Convey("It can use a channel for errors", func() {
// 			err := task.RunWithPipeIO()
// 			if err != nil {
// 				t.Fatal(err)
// 			}

// 			task.In <- `foo`
// 			err = <-task.Errors
// 			So(err.Error(), ShouldContainSubstring, "undefined")
// 		})
// 	})
// }
