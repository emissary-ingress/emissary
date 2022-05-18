package kubeapply

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/derror"
	"github.com/datawire/dlib/dlog"
)

// errorDeadlineExceeded is returned from YAMLCollection.applyAndWait
// if the deadline is exceeded.
var errorDeadlineExceeded = errors.New("timeout exceeded")

// Kubeapply applies the supplied manifests to the kubernetes cluster
// indicated via the kubeinfo argument.  If kubeinfo is nil, it will
// look in the standard default places for cluster configuration.  If
// any phase takes longer than perPhaseTimeout to become ready, then
// it returns early with an error.
func Kubeapply(ctx context.Context, kubeclient *kates.Client, perPhaseTimeout time.Duration, debug, dryRun bool, files ...string) error {
	collection, err := CollectYAML(files...)
	if err != nil {
		return fmt.Errorf("CollectYAML: %w", err)
	}

	if err = collection.ApplyAndWait(ctx, kubeclient, perPhaseTimeout, debug, dryRun); err != nil {
		return fmt.Errorf("ApplyAndWait: %w", err)
	}

	return nil
}

// A YAMLCollection is a collection of YAML files to later be applied.
type YAMLCollection map[string][]string

// CollectYAML takes several file or directory paths, and returns a
// collection of the YAML files in them.
func CollectYAML(paths ...string) (YAMLCollection, error) {
	ret := make(YAMLCollection)
	for _, path := range paths {
		err := filepath.Walk(path, func(filename string, fileinfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fileinfo.IsDir() {
				return nil
			}

			if strings.HasSuffix(filename, ".yaml") {
				ret.addFile(filename)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func hasNumberPrefix(filepart string) bool {
	if len(filepart) < 3 {
		return false
	}
	return '0' <= filepart[0] && filepart[0] <= '9' &&
		'0' <= filepart[1] && filepart[1] <= '9' &&
		filepart[2] == '-'
}

func (collection YAMLCollection) addFile(path string) {
	_, notdir := filepath.Split(path)
	phaseName := "last" // all letters sort after all numbers; "last" is after all numbered phases
	if hasNumberPrefix(notdir) {
		phaseName = notdir[:2]
	}

	collection[phaseName] = append(collection[phaseName], path)
}

// ApplyAndWait applies the collection of YAML, and waits for all
// Resources described in it to be ready.  If any phase takes longer
// than perPhaseTimeout to become ready, then it returns early with an
// error.
func (collection YAMLCollection) ApplyAndWait(
	ctx context.Context,
	kubeclient *kates.Client,
	perPhaseTimeout time.Duration,
	debug, dryRun bool,
) error {
	phaseNames := make([]string, 0, len(collection))
	for phaseName := range collection {
		phaseNames = append(phaseNames, phaseName)
	}
	sort.Strings(phaseNames)

	for _, phaseName := range phaseNames {
		// Note: applyAndWait takes a separate 'deadline' argument, rather than the
		// implicitly using `context.WithDeadline`, so that we can detect whether it's our
		// per-phase timeout that triggered, or a broader "everything" timeout on the
		// Context.
		deadline := time.Now().Add(perPhaseTimeout)
		if err := applyAndWait(ctx, kubeclient, deadline, debug, dryRun, collection[phaseName]); err != nil {
			if errors.Is(err, errorDeadlineExceeded) {
				err = fmt.Errorf("phase %q not ready after %v: %w", phaseName, perPhaseTimeout, err)
			}
			return err
		}
	}
	return nil
}

func applyAndWait(ctx context.Context, kubeclient *kates.Client, deadline time.Time, debug, dryRun bool, sourceFilenames []string) error {
	expandedFilenames, err := expand(ctx, sourceFilenames)
	if err != nil {
		return fmt.Errorf("expanding YAML: %w", err)
	}

	waiter, err := NewWaiter(kubeclient)
	if err != nil {
		return err
	}

	valid := make(map[string]bool)
	var scanErrs derror.MultiError
	for _, filename := range expandedFilenames {
		valid[filename] = true
		if err := waiter.Scan(ctx, filename); err != nil {
			scanErrs = append(scanErrs, fmt.Errorf("watch %q: %w", filename, err))
			valid[filename] = false
		}
	}
	if !debug {
		// Unless the debug flag is on, clean up temporary expanded files when we're
		// finished.
		defer func() {
			for _, filename := range expandedFilenames {
				if valid[filename] {
					if err := os.Remove(filename); err != nil {
						// os.Remove returns an *io/fs.PathError that
						// already includes the filename; no need for us to
						// explicitly include the filename in the log line.
						dlog.Error(ctx, err)
					}
				}
			}
		}()
	}
	if len(scanErrs) > 0 {
		return fmt.Errorf("waiter: %w", scanErrs)
	}

	if err := kubectlApply(ctx, kubeclient, dryRun, expandedFilenames); err != nil {
		return err
	}

	finished, err := waiter.Wait(ctx, deadline)
	if err != nil {
		return err
	}
	if !finished {
		return errorDeadlineExceeded
	}

	return nil
}

func expand(ctx context.Context, names []string) ([]string, error) {
	dlog.Printf(ctx, "expanding %s\n", strings.Join(names, " "))
	var result []string
	for _, n := range names {
		resources, err := LoadResources(ctx, n)
		if err != nil {
			return nil, err
		}
		out := n + ".o"
		err = SaveResources(out, resources)
		if err != nil {
			return nil, err
		}
		result = append(result, out)
	}
	return result, nil
}

func kubectlApply(ctx context.Context, kubeclient *kates.Client, dryRun bool, filenames []string) error {
	stdio := kates.IOStreams{
		In:     nil,
		Out:    dlog.StdLogger(ctx, dlog.LogLevelInfo).Writer(),
		ErrOut: dlog.StdLogger(ctx, dlog.LogLevelWarn).Writer(),
	}

	var args []string
	if dryRun {
		args = append(args, "--dry-run")
	}
	for _, filename := range filenames {
		// flock(2) each file that we're passing to `kubectl apply`.
		// https://github.com/datawire/teleproxy/issues/77
		filehandle, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer filehandle.Close()
		if err := syscall.Flock(int(filehandle.Fd()), syscall.LOCK_EX); err != nil {
			return err
		}

		// pass the file to `kubectl apply`
		args = append(args, "-f", filename)
	}

	if err := kubeclient.IncoherentApply(ctx, stdio, args...); err != nil {
		return err
	}
	return nil
}
