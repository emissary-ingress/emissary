package helm

import (
	"fmt"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/helm/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NOTE: these methods are mostly interesting for the Operator

// GetReleaseMgrForInstance returns a Helm release manager for an instance
func (lc *HelmDownloader) GetReleaseMgrForInstance(ins *unstructured.Unstructured, values map[string]string) (release.Manager, error) {
	if lc.downDir == "" {
		panic(fmt.Errorf("no chart directory: must Download() before creating a manager"))
	}

	mgr := lc.mgr
	if mgr == nil {
		// Get a config to talk to the apiserver
		cfg, err := config.GetConfig()
		if err != nil {
			lc.log.Printf("%w", err)
			return nil, err
		}

		// Create a new Cmd to provide shared dependencies and start components
		mgr, err = manager.New(cfg, manager.Options{Namespace: ins.GetNamespace()})
		if err != nil {
			return nil, err
		}
	}

	factory := release.NewManagerFactory(mgr, filepath.Dir(lc.downChartFile))
	chartMgr, err := factory.NewManager(ins, values)
	if err != nil {
		return nil, err
	}

	return chartMgr, nil
}
