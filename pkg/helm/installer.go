package helm

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

func (lc *HelmDownloader) Install(namespace string, values map[string]interface{}) (*release.Release, error) {
	// based on https://github.com/helm/helm/blob/master/cmd/helm/install.go#L158-L225

	settings := cli.New()
	// TODO: we should override:
	//	     KubeConfig string     // KubeConfig is the path to the kubeconfig file
	//	     KubeContext string    // KubeContext is the name of the kubeconfig context.

	//cf := genericclioptions.NewConfigFlags(true)
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace,
		"secrets", lc.log.Printf); err != nil {
		return nil, err
	}

	client := action.NewInstall(actionConfig)
	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}
	client.ReleaseName = "ambassador"
	client.Namespace = namespace
	client.Wait = true
	client.Atomic = true // installation process purges chart on fail

	chartRequested, err := loader.Load(filepath.Dir(lc.downChartFile))
	if err != nil {
		return nil, err
	}

	for k, v := range values {
		lc.log.Printf("Using helm chart value: %s=%s", k, v)
	}

	oldOutput := log.Writer() // save the old output and set lc.log as the logs writer
	log.SetOutput(lc.log.Writer())

	notFoundErr := func(err error) bool {
		return err != nil && strings.Contains(err.Error(), "not found")
	}

	release, err := client.Run(chartRequested, values)
	if err != nil {
		uninstall := action.NewUninstall(actionConfig)
		_, uninstallErr := uninstall.Run("ambassador")

		// In certain cases, InstallRelease will return a partial release in
		// the response even when it doesn't record the release in its release
		// store (e.g. when there is an error rendering the release manifest).
		// In that case the rollback will fail with a not found error because
		// there was nothing to rollback.
		//
		// Only log a message about a rollback failure if the failure was caused
		// by something other than the release not being found.
		if uninstallErr != nil && !notFoundErr(uninstallErr) {
			return nil, fmt.Errorf("failed installation (%s) and failed rollback: %w", err, uninstallErr)
		}
	}
	log.SetOutput(oldOutput) // restore the old output

	return release, err
}
