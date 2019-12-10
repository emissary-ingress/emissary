package content

import (
	"io/ioutil"
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
	ReadFileBytes(path string) ([]byte, error)
}

type BetterFS struct {
	billy.Filesystem
}

func JustName(fn string) string {
	return strings.TrimSuffix(fn, filepath.Ext(fn))
}

func NewLocalDir(path string) *BetterFS {
	return &BetterFS{osfs.New(path)}
}

func (bfs *BetterFS) Fs() Globbable {
	return bfs
}

func NewChroot(fs Globbable, path string) (*BetterFS, error) {
	chroot, err := fs.Chroot(path)
	if err != nil {
		return nil, err
	}
	return &BetterFS{chroot}, nil
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
