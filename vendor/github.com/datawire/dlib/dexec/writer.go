package dexec

import (
	"bytes"
	"io"
	"os"
)

func fixupWriter(o io.Writer, log func(error, []byte)) io.Writer {
	if o == nil {
		o = nilWriter{}
	}
	if _, isFile := o.(*os.File); isFile {
		return o
	}
	o = &loggingWriter{
		log:    log,
		writer: o,
	}
	return o
}

type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) { return len(p), nil }

type loggingWriter struct {
	log    func(error, []byte)
	writer io.Writer
}

func (l *loggingWriter) Write(p []byte) (n int, err error) {
	toLog := p
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

	n, err = l.writer.Write(p)
	if err != nil {
		l.log(err, nil)
	}
	return n, err
}
