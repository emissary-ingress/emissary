package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/lyft/protoc-gen-star"
)

type PrinterModule struct {
	*pgs.ModuleBase
}

func ASTPrinter() *PrinterModule { return &PrinterModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *PrinterModule) Name() string { return "printer" }

func (p *PrinterModule) Execute(pkg pgs.Package, pkgs map[string]pgs.Package) []pgs.Artifact {
	p.PushDir(pkg.Files()[0].OutputPath().Dir().String())
	defer p.Pop()
	p.Debug("printing:", pkg.GoName())

	buf := &bytes.Buffer{}
	v := initPrintVisitor(buf, "")
	p.CheckErr(pgs.Walk(v, pkg), "unable to print AST tree")

	if ok, _ := p.Parameters().Bool("log_tree"); ok {
		p.Logf("Proto Tree:\n%s", buf.String())
	}

	p.AddGeneratorFile(
		p.JoinPath(pkg.GoName().LowerSnakeCase().String()+".tree.txt"),
		buf.String(),
	)

	return p.Artifacts()
}

const (
	startNodePrefix = "┳ "
	subNodePrefix   = "┃"
	leafNodePrefix  = "┣"
	leafNodeSpacer  = "━ "
)

type PrinterVisitor struct {
	pgs.Visitor
	prefix string
	w      io.Writer
}

func initPrintVisitor(w io.Writer, prefix string) pgs.Visitor {
	v := PrinterVisitor{
		prefix: prefix,
		w:      w,
	}
	v.Visitor = pgs.PassThroughVisitor(&v)
	return v
}

func (v PrinterVisitor) leafPrefix() string {
	if strings.HasSuffix(v.prefix, subNodePrefix) {
		return strings.TrimSuffix(v.prefix, subNodePrefix) + leafNodePrefix
	}
	return v.prefix
}

func (v PrinterVisitor) writeSubNode(str string) pgs.Visitor {
	fmt.Fprintf(v.w, "%s%s%s\n", v.leafPrefix(), startNodePrefix, str)
	return initPrintVisitor(v.w, fmt.Sprintf("%s%v", v.prefix, subNodePrefix))
}

func (v PrinterVisitor) writeLeaf(str string) {
	fmt.Fprintf(v.w, "%s%s%s\n", v.leafPrefix(), leafNodeSpacer, str)
}

func (v PrinterVisitor) VisitPackage(p pgs.Package) (pgs.Visitor, error) {
	return v.writeSubNode("Package: " + p.GoName().String()), nil
}

func (v PrinterVisitor) VisitFile(f pgs.File) (pgs.Visitor, error) {
	return v.writeSubNode("File: " + f.Name().String()), nil
}

func (v PrinterVisitor) VisitMessage(m pgs.Message) (pgs.Visitor, error) {
	return v.writeSubNode("Message: " + m.Name().String()), nil
}

func (v PrinterVisitor) VisitEnum(e pgs.Enum) (pgs.Visitor, error) {
	return v.writeSubNode("Enum: " + e.Name().String()), nil
}

func (v PrinterVisitor) VisitService(s pgs.Service) (pgs.Visitor, error) {
	return v.writeSubNode("Service: " + s.Name().String()), nil
}

func (v PrinterVisitor) VisitEnumValue(ev pgs.EnumValue) (pgs.Visitor, error) {
	v.writeLeaf(ev.Name().String())
	return nil, nil
}

func (v PrinterVisitor) VisitField(f pgs.Field) (pgs.Visitor, error) {
	v.writeLeaf(f.Name().String())
	return nil, nil
}

func (v PrinterVisitor) VisitMethod(m pgs.Method) (pgs.Visitor, error) {
	v.writeLeaf(m.Name().String())
	return nil, nil
}
