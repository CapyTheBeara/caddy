package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/fsnotify.v1"
)

var zeroTime = time.Now()

var mapStringOp = map[string]fsnotify.Op{
	"create": fsnotify.Create,
	"write":  fsnotify.Write,
	"remove": fsnotify.Remove,
	"rename": fsnotify.Rename,
	"chmod":  fsnotify.Chmod,
}

func NewWatcher(c *Config) *Watcher {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	if c.EventCutoff == 0 {
		c.EventCutoff = 0.01
	}

	w := Watcher{
		Watcher: fsw,
		Config:  *c,
		Ready:   make(chan bool),
		Events:  make(chan fsnotify.Event),
		Errors:  make(chan error),
		events:  make(map[string]time.Time),
	}

	w.watchDir()
	go w.listen()

	return &w
}

type Config struct {
	Dir, Ext    string
	FileNames   []string
	IgnoreModes []string
	ExcludeDirs []string
	EventCutoff float64
}

type Watcher struct {
	*fsnotify.Watcher
	Config
	IgnoreEvents bool
	Ready        chan bool
	Events       chan fsnotify.Event
	Errors       chan error
	events       map[string]time.Time
}

func (w *Watcher) IsWatchingDir(dir string) bool {
	for _, excl := range w.ExcludeDirs {
		abs, err := filepath.Abs(filepath.Join(w.Dir, excl))
		if err != nil {
			log.Println("[error] Unable to get absolute path", err)
			return false
		}
		if dir == abs {
			return false
		}
	}
	return true
}

func (w *Watcher) watchDir() {
	if len(w.FileNames) == 0 {
		filepath.Walk(w.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatalln("[error] Couldn't watch folder:", err)
			}

			if info.IsDir() {

				hit := false
				for _, d := range w.ExcludeDirs {
					dir := filepath.Join(w.Dir, d)
					if strings.HasPrefix(path, dir) {
						hit = true
						return filepath.SkipDir
					}
				}

				if !hit {
					w.Add(path)
				}
			}
			return nil
		})
		return
	}

	for _, file := range w.FileNames {
		path := filepath.Join(w.Dir, file)
		err := w.Add(filepath.Dir(path))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (w *Watcher) listen() {
	close(w.Ready)

	for {
		select {
		case e := <-w.Watcher.Events:

			/////////////////////////////
			// log.Println(e)

			if w.isDuplicateEvent(e) {
				continue
			}

			if w.shouldIgnore(e) {
				continue
			}

			if w.isNewDir(e) {
				if len(w.FileNames) == 0 {
					w.Add(e.Name)
				}
				continue
			}

			if w.isWatchingEvent(e) {
				go func() {
					w.Events <- e
				}()
			}
		case err := <-w.Watcher.Errors:
			go func() {
				w.Errors <- err
			}()
		}
	}
}

func (w *Watcher) isNewDir(e fsnotify.Event) bool {
	if e.Op == fsnotify.Create {
		fi, err := os.Stat(e.Name)
		if err != nil {
			log.Fatal("[error] Unable to get file info:", err)
		}

		if fi.IsDir() {
			return true
		}
	}
	return false
}

func (w *Watcher) shouldIgnore(e fsnotify.Event) bool {
	if w.IgnoreEvents {
		return true
	}

	if len(w.IgnoreModes) == 0 {
		return false
	}

	for _, field := range w.IgnoreModes {
		if e.Op == mapStringOp[field] {
			return true
		}
	}
	return false
}

func (w *Watcher) isWatchingEvent(e fsnotify.Event) bool {
	if len(w.FileNames) == 0 {
		if w.Ext == "" {
			return true
		}

		return filepath.Ext(e.Name) == "."+w.Ext
	}

	for _, name := range w.FileNames {
		if e.Name == filepath.Join(w.Dir, name) {
			return true
		}
	}
	return false
}

func (w *Watcher) isDuplicateEvent(e fsnotify.Event) bool {
	now := time.Now()
	prev := w.events[e.String()]
	ignore := prev != zeroTime && now.Sub(prev).Seconds() < w.EventCutoff
	w.events[e.String()] = now
	return ignore
}
