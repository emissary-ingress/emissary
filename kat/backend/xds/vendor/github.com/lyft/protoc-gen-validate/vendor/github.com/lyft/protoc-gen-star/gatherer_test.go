package pgs

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/stretchr/testify/assert"
)

func initTestGatherer(t *testing.T) *gatherer {
	gen := generator.New()
	g := &gatherer{PluginBase: &PluginBase{}}
	g.Init(gen)
	return g
}

func TestGatherer_Generate(t *testing.T) {
	t.Parallel()

	f := &generator.FileDescriptor{
		FileDescriptorProto: &descriptor.FileDescriptorProto{
			Name:    proto.String("file.proto"),
			Package: proto.String("pkg"),
		},
	}

	g := initTestGatherer(t)
	gen := generator.New()
	gen.Request.FileToGenerate = []string{f.GetName()}
	g.Generator = Wrap(gen)
	pgg := initGathererPGG(g)
	pgg.name = "pkg"

	g.Generate(f)

	assert.Len(t, g.targets, 1)
	assert.Equal(t, "pkg", g.targets["pkg"].GoName().String())
	assert.Len(t, g.pkgs, 1)
	assert.Equal(t, g.targets["pkg"], g.pkgs[g.targets["pkg"].GoName().String()])
	assert.Len(t, g.targets["pkg"].Files(), 1)

	assert.Equal(t, g.targets["pkg"], g.hydratePackage(f, map[string]string{}))
}

func TestGatherer_HydrateFile(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	pgg := initGathererPGG(g)

	typ := StringT.Proto()

	me := &descriptor.DescriptorProto{
		Name:    proto.String("MapEntry"),
		Options: &descriptor.MessageOptions{MapEntry: proto.Bool(true)},
		Field: []*descriptor.FieldDescriptorProto{
			{
				Name:     proto.String("map_entry_field"),
				Type:     &typ,
				TypeName: proto.String("string"),
			},
		},
	}

	m := &descriptor.DescriptorProto{
		Name: proto.String("Msg"),
		Field: []*descriptor.FieldDescriptorProto{
			{
				Name:     proto.String("msg_field"),
				Type:     &typ,
				TypeName: proto.String("string"),
			},
		},
		NestedType: []*descriptor.DescriptorProto{me},
	}

	e := &descriptor.EnumDescriptorProto{Name: proto.String("Enum")}

	s := &descriptor.ServiceDescriptorProto{Name: proto.String("Svc")}

	df := dummyFile()
	desc := df.Descriptor()
	desc.MessageType = []*descriptor.DescriptorProto{m}
	desc.EnumType = []*descriptor.EnumDescriptorProto{e}
	desc.Service = []*descriptor.ServiceDescriptorProto{s}

	pkg := df.Package()

	comments := map[string]string{}

	pgg.objs = map[string]generator.Object{
		df.lookupName() + ".Msg":          &generator.Descriptor{DescriptorProto: m},
		df.lookupName() + ".Msg.MapEntry": &generator.Descriptor{DescriptorProto: me},
		df.lookupName() + ".Enum":         &generator.EnumDescriptor{EnumDescriptorProto: e},
	}

	f := g.hydrateFile(pkg, desc, comments)
	assert.Equal(t, pkg, f.Package())
	assert.Equal(t, desc, f.Descriptor())
	assert.Equal(t, goFileName(desc), f.OutputPath().String())
	assert.Len(t, f.AllMessages(), 1)
	assert.Len(t, f.Enums(), 1)
	assert.Len(t, f.Services(), 1)

	_, ok := g.seen(f)
	assert.True(t, ok)
	assert.Equal(t, f, g.hydrateFile(pkg, desc, comments))
}

func TestGatherer_HydrateFile_PackageMismatch(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	initGathererPGG(g)
	md := newMockDebugger(t)
	g.BuildContext = Context(md, Parameters{}, "")

	df := dummyFile()
	dp := df.Package()
	desc := df.Descriptor()
	desc.Package = proto.String("not_the_same_as_dp")

	g.hydrateFile(dp, desc, map[string]string{})
	assert.True(t, md.failed)
}

func TestGatherer_HydrateMessage(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	pgg := initGathererPGG(g)

	me := &descriptor.DescriptorProto{
		Name:    proto.String("MapEntry"),
		Options: &descriptor.MessageOptions{MapEntry: proto.Bool(true)},
	}

	nm := &descriptor.DescriptorProto{
		Name: proto.String("NestedMsg"),
	}

	de := dummyEnum()
	ne := de.Descriptor().EnumDescriptorProto

	fld := dummyField().Descriptor()

	o := dummyOneof().Descriptor()

	dm := dummyMsg()
	desc := dm.rawDesc
	desc.Field = []*descriptor.FieldDescriptorProto{fld}
	desc.EnumType = []*descriptor.EnumDescriptorProto{ne}
	desc.NestedType = []*descriptor.DescriptorProto{nm, me}
	desc.OneofDecl = []*descriptor.OneofDescriptorProto{o}

	f := dm.File()

	pgg.objs = map[string]generator.Object{
		lookupName(f, dm):              dm.Descriptor(),
		lookupName(dm, de):             de.Descriptor(),
		dm.lookupName() + ".NestedMsg": &generator.Descriptor{DescriptorProto: nm},
		dm.lookupName() + ".MapEntry":  &generator.Descriptor{DescriptorProto: me},
	}

	m := g.hydrateMessage(f, desc)
	assert.Equal(t, dm.Descriptor(), m.Descriptor())
	assert.Equal(t, f, m.Parent())
	assert.Len(t, m.Enums(), 1)
	assert.Len(t, m.Messages(), 1)
	assert.Len(t, m.MapEntries(), 1)
	assert.Len(t, m.Fields(), 1)
	assert.Len(t, m.OneOfs(), 1)

	_, ok := g.seen(m)
	assert.True(t, ok)
	assert.Equal(t, m, g.hydrateMessage(f, desc))
}

func TestGatherer_HydrateField(t *testing.T) {
	t.Parallel()

	df := dummyField()
	desc := df.Descriptor()
	m := dummyMsg()

	g := initTestGatherer(t)

	f := g.hydrateField(m, desc)
	assert.Equal(t, desc, f.Descriptor())
	assert.Equal(t, m, f.Message())

	_, ok := g.seen(f)
	assert.True(t, ok)
	assert.Equal(t, f, g.hydrateField(m, desc))
}

func TestGatherer_HydrateFieldType_Scalar(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)

	typ := StringT.Proto()
	fld := &field{
		msg: dummyMsg(),
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("scalar"),
			Type:     &typ,
			TypeName: proto.String("*string"),
		},
	}

	g.add(fld)

	ft := g.hydrateFieldType(fld)
	assert.Equal(t, "*string", ft.Name().String())
}

func TestGatherer_HydrateFieldType_Enum(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	pgg := initGathererPGG(g)

	emb := &enum{
		parent:  dummyFile(),
		rawDesc: &descriptor.EnumDescriptorProto{Name: proto.String("EmbeddedEnum")},
	}
	emb.genDesc = &generator.EnumDescriptor{EnumDescriptorProto: emb.rawDesc}

	typ := EnumT.Proto()
	fld := &field{
		msg: dummyMsg(),
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("enum"),
			Type:     &typ,
			TypeName: proto.String("EmbeddedEnum"),
		},
	}

	g.add(emb)
	g.add(emb.File())
	g.add(fld)

	pgg.types[fld.desc.GetName()] = fld.desc.GetTypeName()
	pgg.objs[fld.desc.GetTypeName()] = &mockObject{
		file: emb.File().Descriptor().FileDescriptorProto,
		name: []string{emb.Name().String()},
	}

	ft := g.hydrateFieldType(fld)
	assert.True(t, ft.IsEnum())
	assert.Equal(t, "EmbeddedEnum", ft.Name().String())
}

func TestGatherer_HydrateFieldType_Embed(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	pgg := initGathererPGG(g)

	emb := &msg{
		parent:  dummyFile(),
		rawDesc: &descriptor.DescriptorProto{Name: proto.String("EmbeddedMessage")},
	}
	emb.genDesc = &generator.Descriptor{DescriptorProto: emb.rawDesc}

	typ := MessageT.Proto()
	fld := &field{
		msg: dummyMsg(),
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("embeded"),
			Type:     &typ,
			TypeName: proto.String("*EmbeddedMessage"),
		},
	}

	g.add(emb)
	g.add(emb.File())
	g.add(fld)

	pgg.types[fld.desc.GetName()] = fld.desc.GetTypeName()
	pgg.objs[fld.desc.GetTypeName()] = &mockObject{
		file: emb.File().Descriptor().FileDescriptorProto,
		name: []string{emb.Name().String()},
	}

	ft := g.hydrateFieldType(fld)
	assert.True(t, ft.IsEmbed())
	assert.Equal(t, "*EmbeddedMessage", ft.Name().String())
}

func TestGatherer_HydrateFieldType_Group(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	initGathererPGG(g)

	d := newMockDebugger(t)
	g.PluginBase.BuildContext = Context(d, Parameters{}, ".")

	typ := GroupT.Proto()
	fld := &field{
		msg: dummyMsg(),
		desc: &descriptor.FieldDescriptorProto{
			Name: proto.String("deprecated_group"),
			Type: &typ,
		},
	}

	g.add(fld)
	g.hydrateFieldType(fld)
	assert.True(t, d.failed)
}

func TestGatherer_HydrateFieldType_RepeatedScalar(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)

	lbl := Repeated.Proto()
	typ := StringT.Proto()
	fld := &field{
		msg: dummyMsg(),
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("scalar_repeated"),
			Label:    &lbl,
			Type:     &typ,
			TypeName: proto.String("[]string"),
		},
	}

	g.add(fld)

	ft := g.hydrateFieldType(fld)
	assert.True(t, ft.IsRepeated())
	assert.Equal(t, "[]string", ft.Name().String())
	assert.Equal(t, "string", ft.Element().Name().String())
}

func TestGatherer_HydrateFieldType_RepeatedEnum(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	pgg := initGathererPGG(g)

	el := &enum{
		parent:  dummyFile(),
		rawDesc: &descriptor.EnumDescriptorProto{Name: proto.String("EmbeddedEnum")},
	}
	el.genDesc = &generator.EnumDescriptor{EnumDescriptorProto: el.rawDesc}

	lbl := Repeated.Proto()
	typ := EnumT.Proto()
	fld := &field{
		msg: dummyMsg(),
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("enum_repeated"),
			Label:    &lbl,
			Type:     &typ,
			TypeName: proto.String("[]EmbeddedEnum"),
		},
	}

	g.add(el)
	g.add(el.File())
	g.add(fld)

	pgg.types[fld.desc.GetName()] = fld.desc.GetTypeName()
	pgg.objs[fld.desc.GetTypeName()] = &mockObject{
		file: el.File().Descriptor().FileDescriptorProto,
		name: []string{el.Name().String()},
	}

	ft := g.hydrateFieldType(fld)
	assert.True(t, ft.IsRepeated())
	assert.True(t, ft.Element().IsEnum())
	assert.Equal(t, "[]EmbeddedEnum", ft.Name().String())
	assert.Equal(t, "EmbeddedEnum", ft.Element().Name().String())
}

func TestGatherer_HydrateFieldType_RepeatedEmbed(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	pgg := initGathererPGG(g)

	el := &msg{
		parent:  dummyFile(),
		rawDesc: &descriptor.DescriptorProto{Name: proto.String("EmbeddedMessage")},
	}
	el.genDesc = &generator.Descriptor{DescriptorProto: el.rawDesc}

	lbl := Repeated.Proto()
	typ := MessageT.Proto()
	fld := &field{
		msg: dummyMsg(),
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("embeded_repeated"),
			Label:    &lbl,
			Type:     &typ,
			TypeName: proto.String("[]EmbeddedMessage"),
		},
	}

	g.add(el)
	g.add(el.File())
	g.add(fld)

	pgg.types[fld.desc.GetName()] = fld.desc.GetTypeName()
	pgg.objs[fld.desc.GetTypeName()] = &mockObject{
		file: el.File().Descriptor().FileDescriptorProto,
		name: []string{el.Name().String()},
	}

	ft := g.hydrateFieldType(fld)
	assert.True(t, ft.IsRepeated())
	assert.True(t, ft.Element().IsEmbed())
	assert.Equal(t, "[]EmbeddedMessage", ft.Name().String())
	assert.Equal(t, "EmbeddedMessage", ft.Element().Name().String())
}

func TestGatherer_HydrateFieldType_Map(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	pgg := initGathererPGG(g)

	key := &field{
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("key"),
			TypeName: proto.String("string"),
		},
	}
	key.addType(&scalarT{name: TypeName("string")})

	val := &field{
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("value"),
			TypeName: proto.String("int64"),
		},
	}
	val.addType(&scalarT{name: TypeName("int64")})

	me := &msg{
		parent: dummyFile(),
		rawDesc: &descriptor.DescriptorProto{
			Name:    proto.String("FooBarEntry"),
			Options: &descriptor.MessageOptions{MapEntry: proto.Bool(true)},
		},
	}
	me.genDesc = &generator.Descriptor{DescriptorProto: me.rawDesc}
	me.addField(key)
	me.addField(val)

	lbl := Repeated.Proto()
	typ := MessageT.Proto()
	fld := &field{
		msg: dummyMsg(),
		desc: &descriptor.FieldDescriptorProto{
			Name:     proto.String("map_field"),
			Label:    &lbl,
			Type:     &typ,
			TypeName: proto.String("FooBarEntry"),
		},
	}

	g.add(key)
	g.add(val)
	g.add(me)
	g.add(fld)
	g.add(me.File())

	pgg.types[fld.desc.GetName()] = me.Name().String()
	pgg.objs[fld.desc.GetTypeName()] = &mockObject{
		file: me.File().Descriptor().FileDescriptorProto,
		name: []string{me.Name().String()},
	}

	ft := g.hydrateFieldType(fld)
	assert.True(t, ft.IsMap())
	assert.Equal(t, "map[string]int64", ft.Name().String())
	assert.Equal(t, "string", ft.Key().Name().String())
	assert.Equal(t, "int64", ft.Element().Name().String())
}

func TestGatherer_HydrateOneOf(t *testing.T) {
	t.Parallel()

	do := dummyOneof()
	desc := do.Descriptor()

	m := do.Message()
	m.addField(dummyField())

	f := dummyField()
	f.desc.OneofIndex = proto.Int32(123)
	m.addField(f)

	g := initTestGatherer(t)

	o := g.hydrateOneOf(m, 123, desc)
	assert.Equal(t, desc, o.Descriptor())
	assert.Equal(t, m, o.Message())
	assert.Len(t, o.Fields(), 1)
	assert.Equal(t, f, o.Fields()[0])

	_, ok := g.seen(o)
	assert.True(t, ok)
	assert.Equal(t, o, g.hydrateOneOf(m, 123, desc))
}

func TestGatherer_HydrateEnum(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	pgg := initGathererPGG(g)

	de := dummyEnum()
	pgg.objs[de.lookupName()] = de.genDesc
	p := de.Parent()

	desc := de.rawDesc
	desc.Value = []*descriptor.EnumValueDescriptorProto{{}}

	e := g.hydrateEnum(p, desc)
	assert.Equal(t, de.genDesc, e.Descriptor())
	assert.Equal(t, p, e.Parent())
	assert.Len(t, e.Values(), 1)

	_, ok := g.seen(e)
	assert.True(t, ok)
	assert.Equal(t, e, g.hydrateEnum(p, desc))
}

func TestGatherer_HydrateEnumValue(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	e := dummyEnum()
	desc := &descriptor.EnumValueDescriptorProto{}

	ev := g.hydrateEnumValue(e, desc)
	assert.Equal(t, desc, ev.Descriptor())
	assert.Equal(t, e, ev.Enum())

	_, ok := g.seen(ev)
	assert.True(t, ok)
	assert.Equal(t, ev, g.hydrateEnumValue(e, desc))
}

func TestGatherer_HydrateService(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	io := dummyMsg()
	g.add(io)
	f := dummyFile()

	desc := &descriptor.ServiceDescriptorProto{
		Method: []*descriptor.MethodDescriptorProto{{
			InputType:  proto.String(io.lookupName()),
			OutputType: proto.String(io.lookupName()),
		}},
	}

	s := g.hydrateService(f, desc)
	assert.Equal(t, desc, s.Descriptor())
	assert.Equal(t, f, s.File())
	assert.Len(t, s.Methods(), 1)

	_, ok := g.seen(s)
	assert.True(t, ok)
	assert.Equal(t, s, g.hydrateService(f, desc))
}

func TestGatherer_HydrateMethod(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	io := dummyMsg()
	g.add(io)

	s := dummyService()
	desc := &descriptor.MethodDescriptorProto{
		InputType:  proto.String(io.lookupName()),
		OutputType: proto.String(io.lookupName()),
	}

	m := g.hydrateMethod(s, desc)
	assert.Equal(t, io, m.Input())
	assert.Equal(t, io, m.Output())
	assert.Equal(t, s, m.Service())
	assert.Equal(t, desc, m.Descriptor())

	_, ok := g.seen(m)
	assert.True(t, ok)
	assert.Equal(t, m, g.hydrateMethod(s, desc))
}

func TestGatherer_Name(t *testing.T) {
	t.Parallel()
	g := initTestGatherer(t)
	assert.Equal(t, gathererPluginName, g.Name())
}

func TestGatherer_GenerateImports(t *testing.T) {
	t.Parallel()
	g := initTestGatherer(t)
	assert.NotPanics(t, func() { g.GenerateImports(nil) })
}

func TestGatherer_Init(t *testing.T) {
	t.Parallel()

	gen := &generator.Generator{Request: &plugin_go.CodeGeneratorRequest{}}
	g := &gatherer{PluginBase: &PluginBase{}}

	assert.NotPanics(t, func() { g.Init(gen) })
	assert.Equal(t, gen, g.Generator.Unwrap())
	assert.NotNil(t, g.pkgs)
	assert.NotNil(t, g.entities)
}

func TestGatherer_ResolveLookupName(t *testing.T) {
	t.Parallel()

	f := dummyFile()
	m := dummyMsg()

	g := initTestGatherer(t)
	assert.Equal(t, f.Name().String(), g.resolveLookupName(f))
	assert.Equal(t, m.lookupName(), g.resolveLookupName(m))
}

func TestGatherer_Add(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	m := dummyMsg()
	g.add(m)
	assert.Contains(t, g.entities, g.resolveLookupName(m))
	assert.Equal(t, m, g.entities[g.resolveLookupName(m)])
}

func TestGatherer_SeenName(t *testing.T) {
	t.Parallel()

	m := dummyMsg()
	g := initTestGatherer(t)
	g.entities = map[string]Entity{
		"foo": m,
	}

	e, ok := g.seenName("foo")
	assert.True(t, ok)
	assert.Equal(t, m, e)

	e, ok = g.seenName("bar")
	assert.False(t, ok)
	assert.Nil(t, e)
}

func TestGatherer_Seen(t *testing.T) {
	t.Parallel()

	g := initTestGatherer(t)
	m := dummyMsg()
	g.add(m)

	e, ok := g.seen(m)
	assert.True(t, ok)
	assert.Equal(t, m, e)

	e, ok = g.seen(dummyEnum())
	assert.False(t, ok)
	assert.Nil(t, e)
}

func TestGatherer_SeenObj(t *testing.T) {
	t.Parallel()

	m := dummyMsg()
	o := mockObject{
		file: m.File().Descriptor().FileDescriptorProto,
		name: dummyMsg().Name().Split(),
	}

	g := initTestGatherer(t)
	g.add(m.File())
	g.add(m)

	e, ok := g.seenObj(o)
	assert.True(t, ok)
	assert.Equal(t, m, e)
}

func TestGatherer_NameByPath(t *testing.T) {
	t.Parallel()

	file := &descriptor.FileDescriptorProto{
		Package: proto.String("my.package"),
		Name:    proto.String("file.proto"),
		MessageType: []*descriptor.DescriptorProto{
			&descriptor.DescriptorProto{
				Name: proto.String("MyMessage"),
				Field: []*descriptor.FieldDescriptorProto{
					&descriptor.FieldDescriptorProto{Name: proto.String("my_field")},
					&descriptor.FieldDescriptorProto{Name: proto.String("my_oneof_field")},
				},
				NestedType: []*descriptor.DescriptorProto{
					&descriptor.DescriptorProto{Name: proto.String("MyNestedMessage")},
				},
				OneofDecl: []*descriptor.OneofDescriptorProto{
					&descriptor.OneofDescriptorProto{Name: proto.String("my_oneof")},
				},
			},
		},
		EnumType: []*descriptor.EnumDescriptorProto{
			&descriptor.EnumDescriptorProto{
				Name: proto.String("MyEnum"),
				Value: []*descriptor.EnumValueDescriptorProto{
					&descriptor.EnumValueDescriptorProto{Name: proto.String("FIRST")},
					&descriptor.EnumValueDescriptorProto{Name: proto.String("SECOND")},
				},
			},
		},
		Service: []*descriptor.ServiceDescriptorProto{
			&descriptor.ServiceDescriptorProto{
				Name: proto.String("MyService"),
				Method: []*descriptor.MethodDescriptorProto{
					&descriptor.MethodDescriptorProto{Name: proto.String("MyMethod")},
				},
			},
		},
	}

	g := initTestGatherer(t)

	testCases := []struct {
		name string
		path []int32
		want string
	}{
		{
			name: "Package",
			path: []int32{2},
			want: ".my.package",
		},
		{
			name: "Message",
			path: []int32{4, 0},
			want: ".my.package.MyMessage",
		},
		{
			name: "Field in Message",
			path: []int32{4, 0, 2, 0},
			want: ".my.package.MyMessage.my_field",
		},
		{
			name: "OneOf Field in Message",
			path: []int32{4, 0, 2, 1},
			want: ".my.package.MyMessage.my_oneof_field",
		},
		{
			name: "NestedMessage in Message",
			path: []int32{4, 0, 3, 0},
			want: ".my.package.MyMessage.MyNestedMessage",
		},
		{
			name: "OneOf in Message",
			path: []int32{4, 0, 8, 0},
			want: ".my.package.MyMessage.my_oneof",
		},
		{
			name: "Enum",
			path: []int32{5, 0},
			want: ".my.package.MyEnum",
		},
		{
			name: "EnumValue1 in Enum",
			path: []int32{5, 0, 2, 0},
			want: ".my.package.MyEnum.FIRST",
		},
		{
			name: "EnumValue2 in Enum",
			path: []int32{5, 0, 2, 1},
			want: ".my.package.MyEnum.SECOND",
		},
		{
			name: "Service",
			path: []int32{6, 0},
			want: ".my.package.MyService",
		},
		{
			name: "Method in Service",
			path: []int32{6, 0, 2, 0},
			want: ".my.package.MyService.MyMethod",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v, err := g.nameByPath(file, tc.path)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, v)
		})
	}

}

type mockObject struct {
	generator.Object
	file *descriptor.FileDescriptorProto
	name []string
}

func (o mockObject) File() *descriptor.FileDescriptorProto { return o.file }
func (o mockObject) TypeName() []string                    { return o.name }

type mockGathererPGG struct {
	ProtocGenGo
	objs  map[string]generator.Object
	types map[string]string
	name  string
}

func initGathererPGG(g *gatherer) *mockGathererPGG {
	pgg := &mockGathererPGG{
		ProtocGenGo: g.Generator,
		objs:        map[string]generator.Object{},
		types:       map[string]string{},
	}
	g.Generator = pgg
	return pgg
}

func (pgg *mockGathererPGG) GoType(m *generator.Descriptor, f *descriptor.FieldDescriptorProto) (string, string) {
	return pgg.types[f.GetName()], ""
}

func (pgg *mockGathererPGG) ObjectNamed(s string) generator.Object {
	return pgg.objs[s]
}

func (pgg *mockGathererPGG) packageName(fd packageFD) string { return pgg.name }
