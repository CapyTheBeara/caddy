package watcher

import (
	"errors"
	"testing"
	"time"

	utils "github.com/monocle/caddy/testutils"
	"gopkg.in/fsnotify.v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewWatcher(t *testing.T) {
	Convey("Given a dir with files", t, func() {
		dir := "tmp1"
		utils.RemoveDir(t, dir)
		utils.MakeDir(t, dir+"/foo/1")
		utils.MakeDir(t, dir+"/bar")
		utils.MakeFile(t, dir, "foo/index.js", "var foo;\n", 0)
		utils.MakeFile(t, dir, "bar/main.js", "1", 0)
		utils.MakeFile(t, dir, "foo/nope.foo", "nope", 20)

		Convey("A file watcher will detect relavent file changes", func() {
			defer utils.RemoveDir(t, dir)

			w := NewWatcher(&Config{
				FileNames: []string{dir + "/foo/index.js", dir + "/bar/main.js"},
			})
			<-w.Ready

			utils.UpdateFile(t, "tmp1/foo/index.js", "s")
			e := <-w.Events
			So(e.Name, ShouldEqual, "tmp1/foo/index.js")

			utils.UpdateFile(t, "tmp1/bar/main.js", "s")
			e = <-w.Events
			So(e.Name, ShouldEqual, "tmp1/bar/main.js")

			utils.UpdateFile(t, "tmp1/foo/nope.foo", "s")
			time.Sleep(time.Millisecond * 20)
			select {
			case <-w.Events:
				So("Failed - Wrong file received", ShouldBeNil)
			default:
				So("Passed - wrong file not received", ShouldNotBeBlank)
			}
		})

		Convey("A dir watcher will detect relavent file changes", func() {
			defer utils.RemoveDir(t, dir)

			w := NewWatcher(&Config{
				Dir: dir + "/foo",
			})
			<-w.Ready

			utils.UpdateFile(t, "tmp1/foo/index.js", "s")
			e := <-w.Events
			So(e.Name, ShouldEqual, "tmp1/foo/index.js")

			utils.UpdateFile(t, "tmp1/foo/nope.foo", "s")
			e = <-w.Events
			So(e.Name, ShouldEqual, "tmp1/foo/nope.foo")

			utils.UpdateFile(t, "tmp1/bar/main.js", "s")
			time.Sleep(time.Millisecond * 20)
			select {
			case <-w.Events:
				So("Failed - Wrong file received", ShouldBeNil)
			default:
				So("Passed - wrong file not received", ShouldNotBeBlank)
			}

			// test subdir is watched
			utils.MakeFile(t, dir, "foo/1/1.js", "1", 0)
			e = <-w.Events
			So(e.Name, ShouldEqual, "tmp1/foo/1/1.js")

			// test a newly created subdir is watched
			utils.MakeDir(t, dir+"/foo/1/2", 20)
			utils.MakeFile(t, dir, "foo/1/2/2.js", "1", 0)
			e = <-w.Events
			So(e.Name, ShouldEqual, "tmp1/foo/1/2/2.js")
		})

		Convey("A dir + ext watcher will detect relavent file changes", func() {
			defer utils.RemoveDir(t, dir)

			w := NewWatcher(&Config{
				Dir: dir + "/foo",
				Ext: "js",
			})
			<-w.Ready

			utils.UpdateFile(t, "tmp1/foo/index.js", "s")
			e := <-w.Events
			So(e.Name, ShouldEqual, "tmp1/foo/index.js")

			utils.UpdateFile(t, "tmp1/foo/nope.foo", "s")
			time.Sleep(time.Millisecond * 20)
			select {
			case <-w.Events:
				So("Failed - Wrong file received", ShouldBeNil)
			default:
				So("Passed - wrong file not received", ShouldNotBeBlank)
			}

			utils.UpdateFile(t, "tmp1/bar/main.js", "s")
			time.Sleep(time.Millisecond * 20)
			select {
			case <-w.Events:
				So("Failed - Wrong file received", ShouldBeNil)
			default:
				So("Passed - wrong file not received", ShouldNotBeBlank)
			}
		})

		Convey("Errors are received on the Errors channel", func() {
			defer utils.RemoveDir(t, dir)

			w := NewWatcher(&Config{
				Dir: dir,
			})
			<-w.Ready

			w.fsw.Errors <- errors.New("foo")
			err := <-w.Errors
			So(err.Error(), ShouldEqual, "foo")
		})

		Convey("A watcher can ignore specified file modes", func() {
			defer utils.RemoveDir(t, dir)

			w := NewWatcher(&Config{
				Dir:         dir,
				IgnoreModes: []string{"chmod"},
			})
			<-w.Ready

			evt := fsnotify.Event{Name: dir + "/index.js", Op: fsnotify.Chmod}
			w.fsw.Events <- evt
			time.Sleep(time.Millisecond * 20)

			select {
			case <-w.Events:
				So("Fails - shouldn't get an event", ShouldBeNil)
			default:
				So("Passes - no event processed", ShouldNotBeBlank)
			}
		})

		Convey("Events can be ignored via a toggle", func() {
			defer utils.RemoveDir(t, dir)

			w := NewWatcher(&Config{
				Dir: dir,
			})
			w.IgnoreEvents = true
			<-w.Ready

			evt := fsnotify.Event{Name: dir + "/index.js", Op: fsnotify.Chmod}
			w.fsw.Events <- evt
			time.Sleep(time.Millisecond * 20)

			select {
			case <-w.Events:
				So("Fails - shouldn't process the event", ShouldBeNil)
			default:
				So("Passes - event is not processed", ShouldNotBeBlank)
			}

		})
	})
}

func TestEventTiming(t *testing.T) {
	Convey("Given a dir with files", t, func() {
		dir := "tmp2"
		utils.RemoveDir(t, dir)
		utils.MakeDir(t, dir)

		Convey("Multple identical events within 100ms are considered as one", func() {
			defer utils.RemoveDir(t, dir)

			w := NewWatcher(&Config{
				Dir: dir,
			})
			<-w.Ready

			evt := fsnotify.Event{Name: dir + "/index.js", Op: fsnotify.Rename}
			evt2 := fsnotify.Event{Name: dir + "/main.js", Op: fsnotify.Rename}
			evt3 := fsnotify.Event{Name: dir + "/index.js", Op: fsnotify.Chmod}

			w.fsw.Events <- evt
			w.fsw.Events <- evt2
			w.fsw.Events <- evt
			w.fsw.Events <- evt3

			e := <-w.Events
			So(e.Name, ShouldEqual, dir+"/index.js")
			So(e.Op, ShouldEqual, fsnotify.Rename)

			e = <-w.Events
			So(e.Name, ShouldEqual, dir+"/main.js")

			e = <-w.Events
			So(e.Name, ShouldEqual, dir+"/index.js")
			So(e.Op, ShouldEqual, fsnotify.Chmod)
		})

		Convey("Multple identical events > 100ms apart are considered separate", func() {
			defer utils.RemoveDir(t, dir)

			w := NewWatcher(&Config{
				Dir: dir,
			})
			<-w.Ready

			evt := fsnotify.Event{Name: dir + "/index.js", Op: fsnotify.Rename}
			evt2 := fsnotify.Event{Name: dir + "/main.js", Op: fsnotify.Rename}

			w.fsw.Events <- evt
			w.fsw.Events <- evt2
			time.Sleep(time.Millisecond * 100)
			w.fsw.Events <- evt

			e := <-w.Events
			So(e.Name, ShouldEqual, dir+"/index.js")

			e = <-w.Events
			So(e.Name, ShouldEqual, dir+"/main.js")

			e = <-w.Events
			So(e.Name, ShouldEqual, dir+"/index.js")
		})
	})
}
