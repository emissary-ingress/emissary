package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/go-cmp/cmp"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/datawire/dlib/dlog"
)

var ErrDiff = errors.New("file is not formatted")

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s -<w|d> FILES.schema.../\n", os.Args[0])
		os.Exit(2)
	}
	if err := Main(context.Background(), os.Args[1][1], os.Args[2:]...); err != nil {
		if !errors.Is(err, ErrDiff) {
			fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		}
		os.Exit(1)
	}
}

func processFile(ctx context.Context, op byte, filename string) error {
	var err error
	var file *os.File
	switch op {
	case 'd':
		file, err = os.Open(filename)
	case 'w':
		file, err = os.OpenFile(filename, os.O_RDWR, 0)
	}
	if err != nil {
		return err
	}
	defer file.Close()

	inputBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(inputBytes))
	decoder.DisallowUnknownFields()

	var schema apiext.JSONSchemaProps
	if err := decoder.Decode(&schema); err != nil {
		return fmt.Errorf("%q: %w", filename, err)
	}

	outputBytes, err := json.MarshalIndent(schema, "", "    ")
	if err != nil {
		return fmt.Errorf("%q: %w", filename, err)
	}
	outputBytes = append(outputBytes, '\n')

	if bytes.Equal(inputBytes, outputBytes) {
		dlog.Infof(ctx, "File %q is already propperly formatted", filename)
		return nil
	}
	switch op {
	case 'd':
		fmt.Printf("diff a/%[1]s b/%[1]s\n--- a/%[1]s\n+++ b/%[1]s\n%s\n",
			filename,
			cmp.Diff(string(inputBytes), string(outputBytes)))
		return fmt.Errorf("%q: %w", filename, ErrDiff)
	case 'w':

		if _, err := file.Seek(0, 0); err != nil {
			return err
		}
		if err := file.Truncate(0); err != nil {
			return err
		}
		if _, err := file.Write(outputBytes); err != nil {
			return err
		}
		dlog.Infof(ctx, "File %q reformatted", filename)
	}
	return nil
}

func Main(ctx context.Context, op byte, filenames ...string) error {
	diff := false
	for _, filename := range filenames {
		if err := processFile(ctx, op, filename); err != nil {
			if errors.Is(err, ErrDiff) {
				diff = true
			} else {
				return err
			}
		}
	}
	if diff {
		return ErrDiff
	}
	return nil
}
