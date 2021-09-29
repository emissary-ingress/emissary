package envoy

import (
	"bytes"
	"io"
)

type Prefixer struct {
	output   io.Writer
	prefix   []byte
	first    bool
	leftover []byte
}

var _ io.Writer = &Prefixer{}

func NewPrefixer(output io.Writer, prefix []byte) *Prefixer {
	return &Prefixer{output, prefix, true, nil}
}

func (p *Prefixer) Write(b []byte) (int, error) {
	if p.leftover != nil {
		n, err := p.output.Write(p.leftover)
		if n == len(p.leftover) {
			p.leftover = nil
		} else {
			p.leftover = p.leftover[n:]
		}
		if err != nil {
			return 0, err
		}
	}

	munged := bytes.ReplaceAll(b, []byte("\n"), append([]byte("\n"), p.prefix...))
	if p.first {
		p.first = false
		munged = append(p.prefix, munged...)
	}

	n, err := p.output.Write(munged)

	if n < len(munged) {
		p.leftover = munged[n:]
	} else {
		p.output.Write([]byte("\n"))
	}

	return len(b), err
}
