package content

import (
	"html/template"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

type Globbable interface {
	billy.Filesystem
	Glob(dir, pattern string) ([]string, error)
	ReadFile(path string) (string, error)
}

type BetterFS struct {
	billy.Filesystem
}

type Content struct {
	fs      Globbable
	funcMap template.FuncMap
}
type ContentVars interface {
	GetString(key string) (value string)
}

func NewContent(contentURL *url.URL) *Content {
	funcMap := template.FuncMap{
		// The name "inc" is what the function will be called in the template text.
		"isEven": func(i int) bool {
			return i%2 == 0
		},

		"isOdd": func(i int) bool {
			return i%2 != 0
		},
	}

	if contentURL.Scheme == "" || contentURL.Scheme == "file" {
		return &Content{
			fs:      &BetterFS{osfs.New(contentURL.Path)},
			funcMap: funcMap,
		}
	} else {
		panic("TODO")
	}
}

func (c *Content) Get(vars ContentVars) (tmpl *template.Template, err error) {
	logger := log.WithFields(log.Fields{
		"subsystem": "content",
	})
	logger.Info("Getting content")
	tmpl = template.New("root").Funcs(c.funcMap)
	err = c.loadTemplate(tmpl, "layout", "layout.gohtml")
	if err != nil {
		return
	}

	err = c.loadTemplate(tmpl, "landing", "landing.gohtml")
	if err != nil {
		return
	}
	err = c.loadDir(tmpl, "pages")
	if err != nil {
		return
	}
	err = c.loadDir(tmpl, "fragments")
	if err != nil {
		return
	}
	logger.Info("Ready")
	return
}

func (c *Content) loadDir(tmpl *template.Template, dir string) (err error) {
	logger := log.WithFields(log.Fields{
		"subsystem": "content",
		"dir":       dir,
	})
	logger.Info("Scanning")
	files, err := c.fs.Glob(dir, "*.gohtml")
	if err != nil {
		return
	}
	for _, fn := range files {
		err = c.loadTemplate(tmpl, JustName(fn), c.fs.Join(dir, fn))
		if err != nil {
			return
		}
	}
	return
}

func (c *Content) loadTemplate(tmpl *template.Template, name string, fn string) (err error) {
	logger := log.WithFields(log.Fields{
		"subsystem":     "content",
		"template-name": name,
		"file":          fn,
	})
	data, err := c.fs.ReadFile(fn)
	if err != nil {
		logger.Errorln("reading file", err)
		return
	}
	t := tmpl.New(name).Funcs(c.funcMap)
	_, err = t.Parse(data)
	if err != nil {
		logger.Errorln("parsing file", err)
		return
	}
	logger.Infoln("Loaded")
	return
}

func (bfs *BetterFS) ReadFile(fn string) (data string, err error) {
	logger := log.WithFields(log.Fields{
		"subsystem": "fs",
		"file":      fn,
	})
	fd, err := bfs.Open(fn)
	if err != nil {
		logger.Error("opening")
		return
	}
	defer fd.Close()
	bytes, err := ioutil.ReadAll(fd)
	if err != nil {
		logger.Error("reading")
		return
	}
	data = string(bytes)
	logger.Info("Read")
	return
}

func (bfs *BetterFS) Glob(dir, pattern string) (names []string, err error) {
	files, err := bfs.ReadDir(dir)
	if err != nil {
		return
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		var matches bool
		matches, err = filepath.Match(pattern, f.Name())
		if err != nil {
			return
		}
		if matches {
			names = append(names, f.Name())
		}
	}
	return
}

func JustName(fn string) string {
	return strings.TrimSuffix(fn, filepath.Ext(fn))
}
