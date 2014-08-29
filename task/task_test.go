package task

import (
	"bytes"
	"io"
	"testing"

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

func TestSimpleTask(tt *testing.T) {
	Convey("Given a simple task", tt, func() {
		Convey("It can specify that the file name be sent as a command arg", func() {
			t := NewSingleTask(&Opts{
				Args:      "echo {{fileName}}",
				UsePipeIO: true,
			})

			t.In <- []byte("hello")

			var buf bytes.Buffer
			io.Copy(&buf, t.OutPipe)

			<-t.Done()
			So(buf.String(), ShouldEqual, "hello\n")
		})

		Convey("It has a BeforeRun callback", func() {
			t := NewSingleTask(&Opts{
				Args:      "echo before",
				UsePipeIO: true,
			})

			input := ""
			t.BeforeRun = func(b []byte) {
				input = string(b)
			}

			t.In <- []byte("hello")
			<-t.Done()
			So(input, ShouldEqual, "hello")
		})

		Convey("It has an OnSuccess callback", func() {
			t := NewSingleTask(&Opts{
				Args:      "echo success",
				UsePipeIO: true,
			})

			called := false
			t.OnSuccess = func() {
				called = true
			}

			t.In <- []byte("hello")
			<-t.Done()

			So(called, ShouldBeTrue)
		})

		Convey("It has an OnError callback", func() {
			t := NewSingleTask(&Opts{
				Args: "blargh",
			})

			var er error
			t.OnError = func(err error) {
				er = err
			}

			t.In <- []byte("hello")
			<-t.Done()
			So(er.Error(), ShouldContainSubstring, "file not found")
		})

		Convey("It has an OnTimeout callback", func() {
			t := NewSingleTask(&Opts{
				Args:    "sleep 2",
				Timeout: -1,
			})

			called := false
			t.OnTimeout = func() {
				called = true
			}

			t.In <- []byte("hello")
			<-t.Done()
			So(called, ShouldBeTrue)
		})

		Convey("It has an OnDone callback", func() {
			t := NewSingleTask(&Opts{
				Args:      "echo done",
				UsePipeIO: true,
			})

			called := false
			t.OnDone = func() {
				called = true
			}

			t.In <- []byte("hello")
			<-t.Done()
			So(called, ShouldBeTrue)
		})

		Convey("It can prevent a run", func() {
			t := NewSingleTask(&Opts{
				Args:      "echo prevent",
				UsePipeIO: true,
			})
			t.ShouldRun = false

			onsuccess := false
			t.OnSuccess = func() {
				onsuccess = true
			}

			onerr := false
			t.OnError = func(error) {
				onerr = true
			}

			ontimeout := false
			t.OnTimeout = func() {
				ontimeout = true
			}

			ondone := false
			t.OnDone = func() {
				ondone = true
			}

			t.In <- []byte{}

			<-t.Done()
			So(onsuccess, ShouldBeFalse)
			So(onerr, ShouldBeFalse)
			So(ontimeout, ShouldBeFalse)
			So(ondone, ShouldBeTrue)
		})

	})
}

func TestMultiTask(tt *testing.T) {
	Convey("Given a continously running task", tt, func() {
		t := NewMultiTask(&Opts{
			Args: "test_tasks/node.js",
		})

		Convey("It can use a channel for output", func() {
			t.In <- []byte(`console.log('foo z\nbar');`)
			res := <-t.Out
			So(string(res), ShouldEqual, "foo z\nbar")
		})

		Convey("It can use a channel for errors", func() {
			t.In <- []byte(`foo`)
			err := <-t.Errors
			So(err.Error(), ShouldContainSubstring, "undefined")
		})
	})
}

// func TestTasksNotify(t *testing.T) {
// 	Convey("Notify will signal when all tasks are done", t, func() {
// 		t1 := NewSingleTask(&Opts{
// 			Args:    "sleep 1",
// 			Timeout: -1,
// 		})

// 		t1done := false
// 		t1.OnDone = func() {
// 			t1done = true
// 		}

// 		t2 := NewSingleTask(&Opts{
// 			Args:    "sleep 2",
// 			Timeout: -1,
// 		})

// 		t2done := false
// 		t2.OnDone = func() {
// 			t2done = true
// 		}

// 		allDone := make(chan struct{})
// 		tasks := NewTasks([]Task{t1, t2})
// 		tasks.Notify("msg", allDone)

// 		<-allDone

// 		So(t1done, ShouldBeTrue)
// 		So(t2done, ShouldBeTrue)
// 	})
// }
