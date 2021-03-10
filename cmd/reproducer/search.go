package reproducer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// A fileVistior function is invoked with a relative path to the file, the detected contentType of
// the file, the detected encoding of the file, and the content itself.
type fileVisitor func(path, contentType, encoding string, content []byte) error

// The break search error is used as a signal a file visitor can use to abort the search early.
var breakSearch = errors.Errorf("break search")

// The search function abstracts away the details of searching archives for a given file. It expects
// to be invoked pointing to a file or an archive. It will traverse all text files contained in the
// possibly nested archives and invoke the supplied visitor function with detected content type
// info. See the visitor function definition for an explanation of what is passed to the visitor
// function.
func search(filename string, action fileVisitor) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening file %s", filename)
	}
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.Wrapf(err, "reading file %s", filename)
	}
	return search_r(filename, "", content, action)
}

func search_r(base, filename string, content []byte, action fileVisitor) error {
	contentType := http.DetectContentType(content)
	parts := strings.SplitN(contentType, ";", 2)
	encoding := ""
	if len(parts) == 2 {
		contentType = parts[0]
		encoding = parts[1]
	}

	switch contentType {
	case "application/x-gzip":
		zr, err := gzip.NewReader(bytes.NewReader(content))
		if err != nil {
			return errors.Wrapf(err, "ungzipping file contents")
		}
		unzippedContents, err := ioutil.ReadAll(zr)
		if err != nil {
			return errors.Wrapf(err, "reading gzip contents %s", filename)
		}

		return search_r(base, filename, unzippedContents, action)
	case "application/zip":
		zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
		if err != nil {
			return errors.Wrapf(err, "unzipping contents")
		}
		for _, f := range zr.File {
			rc, err := f.Open()
			if err != nil {
				return errors.Wrapf(err, "opening zip entry %s", f.Name)
			}
			entryContents, err := ioutil.ReadAll(rc)
			if err != nil {
				return errors.Wrapf(err, "reading zip entry %s", f.Name)
			}
			err = search_r(base, filepath.Join(filename, f.Name), entryContents, action)
			if err != nil {
				return err
			}
		}

		return nil
	case "text/plain":
		if filename == "" {
			filename = base
		}
		return action(filename, contentType, encoding, content)
	case "application/octet-stream":
		tin := tar.NewReader(bytes.NewReader(content))
		for {
			header, err := tin.Next()
			if err == io.EOF {
				break
			}

			if err != nil {
				return errors.Wrapf(err, "untarring file %s", filename)
			}

			switch header.Typeflag {
			case tar.TypeReg:
				name := header.Name
				entryContents, err := ioutil.ReadAll(tin)
				if err != nil {
					return errors.Wrapf(err, "decoding tar entry %s", name)
				}

				err = search_r(base, filepath.Join(filename, name), entryContents, action)
				if err != nil {
					return err
				}
			}
		}
	default:
		return errors.Errorf("unrecognized content type %s", contentType)
	}

	return nil
}
