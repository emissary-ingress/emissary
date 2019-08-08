package content

import (
	"html"
	"io"
	"strings"

	"github.com/oxtoacart/bpool"

	"gitlab.com/golang-commonmark/markdown"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

type MarkdownRenderer interface {
	Render(src []byte) string
}

type CommonmarkRenderer struct{}

func (c *CommonmarkRenderer) Render(src []byte) (data string) {
	md := markdown.New(
		markdown.Tables(true),
		markdown.HTML(true))
	data = md.RenderToString(src)
	return
}

type BlackfridayRenderer struct{}

func (c *BlackfridayRenderer) Render(src []byte) (data string) {
	bdata := blackfriday.Run(src,
		blackfriday.WithExtensions(blackfriday.CommonExtensions),
		blackfriday.WithRenderer(newQuoteNonMungingHTMLRenderer()))
	data = string(bdata)
	return
}

type QuoteNonMungingHTMLRenderer struct {
	r blackfriday.Renderer
	b *bpool.BufferPool
}

func newQuoteNonMungingHTMLRenderer() *QuoteNonMungingHTMLRenderer {
	return &QuoteNonMungingHTMLRenderer{
		r: blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
			Flags: blackfriday.HTMLFlagsNone,
		}),
		b: bpool.NewBufferPool(10),
	}
}

var htmlTextEscaper = strings.NewReplacer(
	`&`, "&amp;",
	`<`, "&lt;",
	`>`, "&gt;",
)

func (r *QuoteNonMungingHTMLRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	switch node.Type {
	case blackfriday.Text:
		temp := r.b.Get()
		defer r.b.Put(temp)
		ret := r.r.RenderNode(temp, node, entering)
		_, _ = w.Write([]byte(htmlTextEscaper.Replace(html.UnescapeString(temp.String()))))
		return ret
	default:
		return r.r.RenderNode(w, node, entering)
	}
}

func (r *QuoteNonMungingHTMLRenderer) RenderHeader(w io.Writer, ast *blackfriday.Node) {
	r.r.RenderHeader(w, ast)
}

func (r *QuoteNonMungingHTMLRenderer) RenderFooter(w io.Writer, ast *blackfriday.Node) {
	r.r.RenderFooter(w, ast)
}
