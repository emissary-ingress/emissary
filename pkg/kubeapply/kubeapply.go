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

	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/dlib/derror"
	"github.com/datawire/dlib/dexec"
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
func Kubeapply(ctx context.Context, kubeinfo *k8s.KubeInfo, perPhaseTimeout time.Duration, debug, dryRun bool, files ...string) error {
	collection, err := CollectYAML(files...)
	if err != nil {
		return fmt.Errorf("CollectYAML: %w", err)
	}

	if err = collection.ApplyAndWait(ctx, kubeinfo, perPhaseTimeout, debug, dryRun); err != nil {
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
	kubeinfo *k8s.KubeInfo,
	perPhaseTimeout time.Duration,
	debug, dryRun bool,
) error {
	if kubeinfo == nil {
		kubeinfo = k8s.NewKubeInfo("", "", "")
	}

	phaseNames := make([]string, 0, len(collection))
	for phaseName := range collection {
		phaseNames = append(phaseNames, phaseName)
	}
	sort.Strings(phaseNames)

	for _, phaseName := range phaseNames {
		deadline := time.Now().Add(perPhaseTimeout)
		err := applyAndWait(ctx, kubeinfo, deadline, debug, dryRun, collection[phaseName])
		if err != nil {
			if errors.Is(err, errorDeadlineExceeded) {
				err = fmt.Errorf("phase %q not ready after %v: %w", phaseName, perPhaseTimeout, err)
			}
			return err
		}
	}
	return nil
}

func applyAndWait(ctx context.Context, kubeinfo *k8s.KubeInfo, deadline time.Time, debug, dryRun bool, sourceFilenames []string) error {
	expandedFilenames, err := expand(ctx, sourceFilenames)
	if err != nil {
		return fmt.Errorf("expanding YAML: %w", err)
	}

	cli, err := k8s.NewClient(kubeinfo)
	if err != nil {
		return fmt.Errorf("connecting to cluster %v: %w", kubeinfo, err)
	}
	waiter, err := NewWaiter(cli.Watcher())
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

	if err := kubectlApply(ctx, kubeinfo, dryRun, expandedFilenames); err != nil {
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

func kubectlApply(ctx context.Context, info *k8s.KubeInfo, dryRun bool, filenames []string) error {
	args := []string{"apply"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	for _, filename := range filenames {
		// https://github.com/datawire/ambassador/issues/77
		filehandle, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer filehandle.Close()
		if err := syscall.Flock(int(filehandle.Fd()), syscall.LOCK_EX); err != nil {
			return err
		}
		args = append(args, "-f", filename)
	}
	kargs, err := info.GetKubectlArray(args...)
	if err != nil {
		return err
	}
	dlog.Printf(ctx, "kubectl %s\n", strings.Join(kargs, " "))
	/* #nosec */
	if err := dexec.CommandContext(ctx, "kubectl", kargs...).Run(); err != nil {
		return err
	}

	return nil
}
