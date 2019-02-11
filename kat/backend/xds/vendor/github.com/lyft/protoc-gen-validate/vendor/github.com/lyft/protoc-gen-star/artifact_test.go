package pgs

import (
	"testing"

	"text/template"

	"github.com/stretchr/testify/assert"
)

var (
	badArtifactTpl = template.Must(template.New("bad").Parse("{{ .NonExistentField }}"))
	artifactTpl    = template.Must(template.New("foo").Parse("{{ . }}"))
)

func TestGeneratorFile_ProtoFile(t *testing.T) {
	t.Parallel()

	f := GeneratorFile{
		Name:     "..",
		Contents: "bar",
	}

	pb, err := f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.Name = "foo"
	pb, err = f.ProtoFile()
	assert.NoError(t, err)
	assert.Equal(t, f.Name, pb.GetName())
	assert.Equal(t, f.Contents, pb.GetContent())
}

func TestGeneratorTemplateFile_ProtoFile(t *testing.T) {
	t.Parallel()

	f := GeneratorTemplateFile{
		Name: ".",
		TemplateArtifact: TemplateArtifact{
			Template: badArtifactTpl,
			Data:     "bar",
		},
	}

	pb, err := f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.Name = "foo"
	pb, err = f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.Template = artifactTpl
	pb, err = f.ProtoFile()
	assert.NoError(t, err)
	assert.Equal(t, f.Name, pb.GetName())
	assert.Equal(t, "bar", pb.GetContent())
}

func TestGeneratorAppend_ProtoFile(t *testing.T) {
	t.Parallel()

	f := GeneratorAppend{
		FileName: ".",
		Contents: "bar",
	}

	pb, err := f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.FileName = "foo"
	pb, err = f.ProtoFile()
	assert.NoError(t, err)
	assert.Empty(t, pb.GetName())
	assert.Equal(t, f.Contents, pb.GetContent())
}

func TestGeneratorTemplateAppend_ProtoFile(t *testing.T) {
	t.Parallel()

	f := GeneratorTemplateAppend{
		FileName: "/tmp",
		TemplateArtifact: TemplateArtifact{
			Template: badArtifactTpl,
			Data:     "bar",
		},
	}

	pb, err := f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.FileName = "foo"
	pb, err = f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.Template = artifactTpl
	pb, err = f.ProtoFile()
	assert.NoError(t, err)
	assert.Empty(t, pb.GetName())
	assert.Equal(t, "bar", pb.GetContent())
}

func TestGeneratorInjection_ProtoFile(t *testing.T) {
	t.Parallel()

	f := GeneratorInjection{
		FileName:       "..",
		Contents:       "bar",
		InsertionPoint: "baz",
	}

	pb, err := f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.FileName = "foo"
	pb, err = f.ProtoFile()
	assert.NoError(t, err)
	assert.Equal(t, f.FileName, pb.GetName())
	assert.Equal(t, f.Contents, pb.GetContent())
	assert.Equal(t, f.InsertionPoint, pb.GetInsertionPoint())
}

func TestGeneratorTemplateInjection_ProtoFile(t *testing.T) {
	t.Parallel()

	f := GeneratorTemplateInjection{
		FileName:       ".",
		InsertionPoint: "baz",
		TemplateArtifact: TemplateArtifact{
			Template: badArtifactTpl,
			Data:     "bar",
		},
	}

	pb, err := f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.FileName = "foo"
	pb, err = f.ProtoFile()
	assert.Error(t, err)
	assert.Nil(t, pb)

	f.Template = artifactTpl
	pb, err = f.ProtoFile()
	assert.NoError(t, err)
	assert.Equal(t, f.FileName, pb.GetName())
	assert.Equal(t, "bar", pb.GetContent())
	assert.Equal(t, f.InsertionPoint, pb.GetInsertionPoint())
}
