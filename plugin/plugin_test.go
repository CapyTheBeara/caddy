package plugin

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGoPlugin(t *testing.T) {
	Convey("A .go file argument will run 'go run' ", t, func() {
		c := NewPlugin([]string{"test_plugins/foo.go"})
		b, err := c.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		So(string(b), ShouldEqual, "foo")
	})
}

func TestNodePlugin(t *testing.T) {
	Convey("Given a .js file argument", t, func() {
		p := NewPlugin([]string{"test_plugins/foo.js"})

		Convey("It can use a channel for output", func() {
			err := p.RunWithPipeIO()
			if err != nil {
				t.Fatal(err)
			}

			p.In <- `console.log('foo\nbar');`
			res := <-p.Out
			So(res, ShouldEqual, "foo\nbar")
		})

		Convey("It can use a channel for errors", func() {
			err := p.RunWithPipeIO()
			if err != nil {
				t.Fatal(err)
			}

			p.In <- `foo`
			err = <-p.Errors
			So(err.Error(), ShouldContainSubstring, "undefined")
		})
	})
}

// func TestPythonPlugin(t *testing.T) {
// 	Convey("A .py file argument will run 'python' ", t, func() {
// 		p := NewPlugin([]string{"test_plugins/foo.py"})
// 		err := p.RunWithPipeIO()
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		p.In <- `blarg`
// 		res := <-p.Out
// 		So(res, ShouldEqual, "blarg")
// 	})

// }

// func TestRubyPlugin(t *testing.T) {
// 	Convey("A .rb file argument will run 'ruby' ", t, func() {
// 		p := NewPlugin([]string{"test_plugins/foo.rb"})
// 		err := p.RunWithPipeIO()
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		p.In <- `blarg`

// 		res := <-p.Out
// 		So(res, ShouldEqual, "blarg")
// 	})

// }
