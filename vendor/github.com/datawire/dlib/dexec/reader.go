package dexec

import (
	"bytes"
	"io"
	"os"
)

func fixupReader(o io.Reader, log func(error, []byte)) io.Reader {
	if o == nil {
		o = nilReader{}
	}
	if _, isFile := o.(*os.File); isFile {
		return o
	}
	o = &loggingReader{
		log:    log,
		reader: o,
	}
	return o
}

type nilReader struct{}

func (nilReader) Read(_ []byte) (int, error) { return 0, io.EOF }

type loggingReader struct {
	log    func(error, []byte)
	reader io.Reader
}

func (l *loggingReader) Read(p []byte) (n int, err error) {
	n, err = l.reader.Read(p)

	toLog := p[:n]
	for len(toLog) > 0 {
		nl := bytes.IndexByte(toLog, '\n')
		var line []byte
		if nl < 0 {
			line = toLog
			toLog = nil
		} else {
			line = toLog[:nl+1]
			toLog = toLog[nl+1:]
		}
		l.log(nil, line)
	}

	if err != nil {
		l.log(err, nil)
	}

	return n, err
}
