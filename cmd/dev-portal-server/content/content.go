package content

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/golang-commonmark/markdown"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

type Globbable interface {
	billy.Filesystem
	Glob(dir, pattern string) ([]string, error)
	ReadFile(path string) (string, error)
	ReadFileBytes(path string) ([]byte, error)
}

type BetterFS struct {
	billy.Filesystem
}

type Content struct {
	fs      Globbable
	funcMap template.FuncMap
}
type ContentVars interface {
	SetPages(pages []string)
	CurrentPage() (page string)
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
	err = c.loadTemplateHTML(tmpl, "///layout", "layout.gohtml")
	if err != nil {
		return
	}

	err = c.loadTemplateMD(tmpl, "///landing", "landing.gomd")
	if err != nil {
		return
	}
	pages, err := c.loadDirMD(tmpl, "pages")
	if err != nil {
		return
	}
	vars.SetPages(pages)
	_, err = c.loadDirHTML(tmpl, "fragments")
	if err != nil {
		return
	}
	// templates do not allow dynamic redirects so generate a dynamic template
	page := vars.CurrentPage()
	if pages.Contains(page) {
		c.parseTemplate(tmpl, "///page-magic", "*code*", fmt.Sprintf(`{{template "%s" $}}`, page))
	}
	logger.Info("Ready")
	return
}

type templateList []string

func (tmpls templateList) Contains(name string) bool {
	for _, i := range tmpls {
		if name == i {
			return true
		}
	}
	return false
}

func (c *Content) loadDirHTML(tmpl *template.Template, dir string) (templates templateList, err error) {
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
		name := JustName(fn)
		err = c.loadTemplateHTML(tmpl, name, c.fs.Join(dir, fn))
		if err != nil {
			return
		}
		templates = append(templates, name)
	}
	return
}

func (c *Content) loadDirMD(tmpl *template.Template, dir string) (templates templateList, err error) {
	logger := log.WithFields(log.Fields{
		"subsystem": "content",
		"dir":       dir,
	})
	logger.Info("Scanning")
	files, err := c.fs.Glob(dir, "*.gomd")
	if err != nil {
		return
	}
	for _, fn := range files {
		name := JustName(fn)
		err = c.loadTemplateMD(tmpl, name, c.fs.Join(dir, fn))
		if err != nil {
			return
		}
		templates = append(templates, name)
	}
	return
}

func (c *Content) loadTemplateHTML(tmpl *template.Template, name, fn string) (err error) {
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
	return c.parseTemplate(tmpl, name, fn, data)
}

func (c *Content) loadTemplateMD(tmpl *template.Template, name, fn string) (err error) {
	logger := log.WithFields(log.Fields{
		"subsystem":     "content",
		"template-name": name,
		"file":          fn,
	})
	src, err := c.fs.ReadFileBytes(fn)
	if err != nil {
		logger.Errorln("reading file", err)
		return
	}
	md := markdown.New(
		markdown.Tables(true),
		markdown.HTML(true))
	data := md.RenderToString(src)
	debug := fn + ".debughtml"
	fd, err2 := c.fs.Create(debug)
	if err2 == nil {
		defer fd.Close()
		fd.Write([]byte(data))
	}
	err = c.parseTemplate(tmpl, name, debug, data)
	if err != nil {
		debug := fn + ".debugerr"
		fd, err2 := c.fs.Create(debug)
		if err2 == nil {
			defer fd.Close()
			fd.Write([]byte(err.Error()))
			logger.
				WithField("debug", debug).
				Info("Trouble parsing ", err)
		}
	}
	return
}

func (c *Content) parseTemplate(tmpl *template.Template, name, fn, data string) (err error) {
	logger := log.WithFields(log.Fields{
		"subsystem":     "content",
		"template-name": name,
		"file":          fn,
	})
	t := tmpl.New(name).Funcs(c.funcMap)
	_, err = t.Parse(data)
	if err != nil {
		logger.Errorln("parsing file", err)
		return
	}
	logger.Infoln("Loaded")
	return
}

type ReadSeekerCloser interface {
	io.ReadSeeker
	io.Closer
}
type StaticResource struct {
	Name    string
	Modtime time.Time
	Data    ReadSeekerCloser
	io.Closer
}

func (r *StaticResource) Close() (err error) {
	return r.Data.Close()
}

func (c *Content) GetStatic(fn string) (resource *StaticResource, err error) {
	logger := log.WithFields(log.Fields{
		"subsystem": "content",
		"file":      fn,
	})
	stat, err := c.fs.Stat(fn)
	if err != nil {
		logger.Info(err)
		return
	}
	if stat.IsDir() {
		logger.Info("will not serve directory")
		return nil, fmt.Errorf("Will not serve directory %s", fn)
	}
	fd, err := c.fs.Open(fn)
	if err != nil {
		logger.Info(err)
		return
	}
	resource = &StaticResource{
		Name:    fn,
		Modtime: stat.ModTime(),
		Data:    fd,
	}
	logger.Info("Opened")
	return
}

func (bfs *BetterFS) ReadFileBytes(fn string) (bytes []byte, err error) {
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
	bytes, err = ioutil.ReadAll(fd)
	if err != nil {
		logger.Error("reading")
		return
	}
	return
}

func (bfs *BetterFS) ReadFile(fn string) (data string, err error) {
	bytes, err := bfs.ReadFileBytes(fn)
	if err != nil {
		return
	}
	data = string(bytes)
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
