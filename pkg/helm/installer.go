package helm

import (
	"log"
	"path/filepath"

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

	release, err := client.Run(chartRequested, values)

	log.SetOutput(oldOutput) // restore the old output

	return release, err
}
