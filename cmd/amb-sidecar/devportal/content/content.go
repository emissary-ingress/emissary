package content

import (
	"fmt"
	"html/template"
	"io"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type GlobbableView interface {
	Fs() Globbable
}

type Content struct {
	store   GlobbableView
	funcMap template.FuncMap
	md      MarkdownRenderer
	source  *url.URL
	config  ContentConfig
}

type ContentConfig struct {
	Version string
}

type ContentVars interface {
	SetPages(pages []string)
	CurrentPage() (page string)
}

func sanitizedURL(u *url.URL) *url.URL {
	u, _ = u.Parse("") // make a copy
	if u.User != nil {
		u.User = url.User("*redacted*")
	}
	return u
}

func (c *Content) Source() *url.URL {
	ret, _ := c.source.Parse("")
	return ret
}

func IsLocal(contentURL *url.URL) bool {
	return contentURL.Scheme == "" || contentURL.Scheme == "file"
}

func NewContent(contentURL *url.URL, branch, subdir string) (*Content, error) {
	logger := log.WithFields(log.Fields{
		"subsystem":     "content",
		"contentURL":    sanitizedURL(contentURL).String(),
		"contentSubdir": subdir,
		"contentBranch": branch,
	})
	funcMap := template.FuncMap{
		// The name "inc" is what the function will be called in the template text.
		"isEven": func(i int) bool {
			return i%2 == 0
		},

		"isOdd": func(i int) bool {
			return i%2 != 0
		},
	}

	renderer := &BlackfridayRenderer{}

	var err error

	var content *Content
	if IsLocal(contentURL) {
		logger.Info("Loading content from local path")
		content = &Content{
			store:   NewLocalDir(contentURL.Path),
			funcMap: funcMap,
			md:      renderer,
		}
	} else {
		logger.Info("Loading content from git repo")
		opts := CheckoutOptions{
			RepoURL: contentURL,
			Branch:  branch,
		}
		var checkout *Checkout
		checkout, err = NewRepoCheckout(opts)
		if err != nil {
			return nil, err
		}
		content = &Content{
			store:   checkout,
			funcMap: funcMap,
			md:      renderer,
		}
	}
	store, err := NewChroot(content.store.Fs(), subdir)
	if err != nil {
		return nil, err
	}
	content.store = store
	content.source = contentURL

	err = content.loadConfig()
	if err != nil {
		return nil, err
	}

	return content, nil
}

func NewMockContent(config ContentConfig) *Content {
	return &Content{
		config: config,
	}
}

func (c *Content) loadConfig() error {
	f, err := c.Fs().Open("devportal.yaml")
	if err != nil {
		return err
	}
	defer f.Close()
	cfg := ContentConfig{}
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return err
	}
	c.config = cfg
	return nil
}

func (c *Content) Config() ContentConfig {
	return c.config
}

func (c *Content) Fs() Globbable {
	return c.store.Fs()
}

func (c *Content) Get(vars ContentVars) (*template.Template, error) {
	logger := log.WithFields(log.Fields{
		"subsystem": "content",
	})
	logger.Info("Getting content")
	tmpl := template.New("root").Funcs(c.funcMap)
	err := c.loadTemplateHTML(tmpl, "///layout", "layout.gohtml")
	if err != nil {
		return nil, err
	}

	err = c.loadTemplateMD(tmpl, "///landing", "landing.gomd")
	if err != nil {
		return nil, err
	}
	pagePrefix := "page/"
	pages, err := c.loadDirMD(tmpl, "pages", pagePrefix)
	if err != nil {
		return nil, err
	}
	vars.SetPages(pages)
	_, err = c.loadDirHTML(tmpl, "fragments")
	if err != nil {
		return nil, err
	}
	// templates do not allow dynamic redirects so generate a dynamic template
	page := vars.CurrentPage()
	magic := fmt.Sprintf(`{{template "%s%s" $}}`, pagePrefix, page)
	if !pages.Contains(page) {
		magic = `{{template "missing-page" $}}`
	}
	c.parseTemplate(tmpl, "///page-magic", "*code*", magic)
	logger.Info("Ready")
	return tmpl, nil
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
	files, err := c.Fs().Glob(dir, "*.gohtml")
	if err != nil {
		return
	}
	for _, fn := range files {
		name := JustName(fn)
		err = c.loadTemplateHTML(tmpl, name, c.Fs().Join(dir, fn))
		if err != nil {
			return
		}
		templates = append(templates, name)
	}
	return
}

func (c *Content) loadDirMD(tmpl *template.Template, dir string, templatePrefix string) (templates templateList, err error) {
	logger := log.WithFields(log.Fields{
		"subsystem": "content",
		"dir":       dir,
	})
	logger.Info("Scanning")
	files, err := c.Fs().Glob(dir, "*.gomd")
	if err != nil {
		return
	}
	for _, fn := range files {
		name := JustName(fn)
		err = c.loadTemplateMD(tmpl, templatePrefix+name, c.Fs().Join(dir, fn))
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
	data, err := c.Fs().ReadFile(fn)
	if err != nil {
		logger.Errorln("reading file", err)
		return
	}
	return c.parseTemplate(tmpl, name, fn, data)
}

func (c *Content) loadTemplateMD(tmpl *template.Template, name, fn string) error {
	logger := log.WithFields(log.Fields{
		"subsystem":     "content",
		"template-name": name,
		"file":          fn,
	})
	src, err := c.Fs().ReadFileBytes(fn)
	if err != nil {
		logger.Errorln("reading file", err)
		return err
	}

	data := c.md.Render(src)

	debug := fn + ".debughtml"
	fd, err2 := c.Fs().Create(debug)
	if err2 == nil {
		defer fd.Close()
		fd.Write([]byte(data))
	}
	err = c.parseTemplate(tmpl, name, debug, data)
	if err != nil {
		debug := fn + ".debugerr"
		fd, err2 := c.Fs().Create(debug)
		if err2 == nil {
			defer fd.Close()
			fd.Write([]byte(err.Error()))
			logger.
				WithField("debug", debug).
				Info("Trouble parsing ", err)
		}
	}
	return err
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
	stat, err := c.Fs().Stat(fn)
	if err != nil {
		logger.Info(err)
		return
	}
	if stat.IsDir() {
		logger.Info("will not serve directory")
		return nil, fmt.Errorf("Will not serve directory %s", fn)
	}
	fd, err := c.Fs().Open(fn)
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
