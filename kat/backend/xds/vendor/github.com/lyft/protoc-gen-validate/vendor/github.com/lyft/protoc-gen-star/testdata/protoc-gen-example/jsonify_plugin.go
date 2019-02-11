package main

import (
	"text/template"

	"fmt"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/lyft/protoc-gen-star"
)

var (
	marshalJSONTpl   = template.Must(template.New("marshalJSON").Parse(marshalJSON))
	unmarshalJSONTpl = template.Must(template.New("unmarshalJSON").Parse(unmarshalJSON))
)

// JSONifyPlugin adds encoding/json Marshaler and Unmarshaler methods on PB
// messages that utilizes the more correct jsonpb package.
// See: https://godoc.org/github.com/golang/protobuf/jsonpb
type JSONifyPlugin struct {
	*pgs.PluginBase
}

// JSONify returns an initialized JSONifyPlugin
func JSONify() *JSONifyPlugin { return &JSONifyPlugin{&pgs.PluginBase{}} }

// Name satisfies the generator.Plugin interface.
func (p *JSONifyPlugin) Name() string { return "jsonify" }

// Generate satisfies the generator.Plugin interface.
func (p *JSONifyPlugin) Generate(file *generator.FileDescriptor) {
	if !p.BuildTarget(file.GetName()) || len(file.GetMessageType()) == 0 {
		return
	}

	p.Push(file.GetName())
	defer p.Pop()

	jpb := p.AddImport("jsonpb", "github.com/golang/protobuf/jsonpb", nil)
	js := p.AddImport("json", "encoding/json", nil)
	bytes := p.AddImport("bytes", "bytes", nil)

	for _, m := range file.GetMessageType() {
		p.generateMessage(msgData{
			DescriptorProto: m,
			Bytes:           bytes,
			JSON:            js,
			JSONPB:          jpb,
		})
	}
}

func (p *JSONifyPlugin) generateMessage(m msgData) {
	if m.GetOptions().GetMapEntry() {
		return
	}

	p.Push(m.GetName()).Debug("implementing json.Marshaler/Unmarshaler interface")
	defer p.Pop()

	p.C80(m.Name(), "Marshaler describes the default jsonpb.Marshaler used by all instances of ",
		m.Name(), ". This struct is safe to replace or modify but should not be done so concurrently.")
	p.P("var ", m.Name(), "Marshaler = new(", m.JSONPB, ".Marshaler)")

	p.C80("MarshalJSON satisfies the encoding/json Marshaler interface. This method uses the more correct jsonpb package to correctly marshal the message.")
	p.T(marshalJSONTpl, m)

	p.C80(m.Name(), "Unmarshaler describes the default jsonpb.Unmarshaler used by all instances of ",
		m.Name(), ". This struct is safe to replace or modify but should not be done so concurrently.")
	p.P("var ", m.Name(), "Unmarshaler = new(", m.JSONPB, ".Unmarshaler)")

	p.C80("UnmarshalJSON satisfies the encoding/json Unmarshaler interface. This method uses the more correct jsonpb package to correctly unmarshal the message.")
	p.T(unmarshalJSONTpl, m)

	for _, nm := range m.GetNestedType() {
		p.generateMessage(msgData{
			DescriptorProto: nm,
			Parent:          m.Name(),
			Bytes:           m.Bytes,
			JSON:            m.JSON,
			JSONPB:          m.JSONPB})
	}
}

type msgData struct {
	*descriptor.DescriptorProto
	Parent              string
	Bytes, JSON, JSONPB string
}

func (d msgData) Name() string {
	if d.Parent == "" {
		return d.GetName()
	}

	return fmt.Sprintf("%s_%s", d.Parent, d.GetName())
}

const marshalJSON = `func (m *{{ .Name }}) MarshalJSON() ([]byte, error) {
	if m == nil {
		return {{ .JSON }}.Marshal(nil)
	}


	buf := &{{ .Bytes }}.Buffer{}
	if err := {{ .Name }}Marshaler.Marshal(buf, m); err != nil {
	  return nil, err
	}

	return buf.Bytes(), nil
}

var _ {{ .JSON }}.Marshaler = (*{{ .Name }})(nil)
`

const unmarshalJSON = `func (m *{{ .Name }}) UnmarshalJSON(b []byte) error {
	return {{ .Name }}Unmarshaler.Unmarshal({{ .Bytes }}.NewReader(b), m)
}

var _ {{ .JSON }}.Unmarshaler = (*{{ .Name }})(nil)
`

var _ pgs.Plugin = (*JSONifyPlugin)(nil)
