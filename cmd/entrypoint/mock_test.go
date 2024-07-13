package entrypoint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"

	"github.com/otiai10/copy"
)

func TestFakeMock(t *testing.T) {
	mockDir := os.Getenv("MOCK_DIR")

	if mockDir == "" {
		t.Skip("MOCK_DIR not set")
	}

	dirToScan := mockDir

	manifestPath := filepath.Join(mockDir, "manifests")
	fileInfo, err := os.Stat(manifestPath)

	if err == nil && fileInfo.IsDir() {
		dirToScan = manifestPath
	}

	// You can use os.Setenv to set environment variables, and they will affect the test harness.
	// Here, we'll force secret validation, so that invalid Secrets won't get passed all the way
	// to Envoy.
	os.Setenv("AMBASSADOR_FORCE_SECRET_VALIDATION", "true")

	// Use RunFake() to spin up the ambassador control plane with its inputs wired up to the Fake
	// APIs. This will automatically invoke the Setup() method for the Fake and also register the
	// Teardown() method with the Cleanup() hook of the supplied testing.T object.
	//
	// Note that we _must_ set EnvoyConfig true to allow checking IR Features later, even though
	// we don't actually do any checking of the Envoy config in this test.

	outputDir := "/tmp/mock"

	// Make sure outputDir is an empty directory
	err = os.RemoveAll(outputDir)
	if err != nil {
		t.Fatalf("Failed to remove output directory: %v", err)
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	fakeConfig := entrypoint.FakeConfig{
		EnvoyConfig: true,
		OutputDir:   outputDir,
	}

	f := entrypoint.RunFake(t, fakeConfig, nil)

	// Scan dirToScan for subdirectories. We'll do one pass for each
	// subdirectory, or one pass for dirToScan itself if there are no
	// subdirectories.

	dirs := make([]string, 0)
	goldDirs := make([]string, 0)

	dirEntries, err := os.ReadDir(dirToScan)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			dirs = append(dirs, filepath.Join(dirToScan, dirEntry.Name()))
			goldDirs = append(goldDirs, filepath.Join(mockDir, "gold", dirEntry.Name()))
		}
	}

	// If there are no subdirectories, just scan dirToScan itself, and
	// use mockDir/gold as the gold directory.
	if len(dirs) == 0 {
		dirs = append(dirs, dirToScan)
		goldDirs = append(goldDirs, filepath.Join(mockDir, "gold"))
	}

	// OK, we have a list of directories to scan. Let's do it.
	for i, passDir := range dirs {
		t.Logf("Running pass for directory %s", passDir)

		// Scan the directory for YAML files
		files, err := os.ReadDir(passDir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		// Upsert each YAML file found
		for _, file := range files {
			if file.Type().IsRegular() && strings.HasSuffix(file.Name(), ".yaml") {
				t.Logf("Upserting file %s", file.Name())
				filePath := filepath.Join(passDir, file.Name())
				assert.NoError(t, f.UpsertFile(filePath))
			}
		}

		// After we've upserted everything, flush the Fake harness once to
		// generate a configuration for this pass.
		f.Flush()

		entry, err := f.GetSnapshotEntry(func(entry entrypoint.SnapshotEntry) bool {
			return entry.Disposition == entrypoint.SnapshotReady
		})

		require.NoError(t, err)

		// We want the gold files to be in this pass's goldDir.
		goldDir := goldDirs[i]

		// Remove any old goldDir
		err = os.RemoveAll(goldDir)

		if err != nil {
			fmt.Printf("Could not remove %s: %s\n", goldDir, err)
			os.Exit(1)
		}

		// Copy the contents of outputDir to goldDir
		err = copy.Copy(outputDir, goldDir)

		if err != nil {
			fmt.Printf("Could not copy %s to %s: %s\n", outputDir, goldDir, err)
			os.Exit(1)
		}

		// Dump the snapshot and its Envoy config as JSON to a file
		err = f.SaveSnapshotEntry(entry, goldDir)

		if err != nil {
			t.Fatalf("Failed to save snapshot entry: %v", err)
		}

		t.Logf("Envoy config dumped to %s", goldDir)
	}
}
