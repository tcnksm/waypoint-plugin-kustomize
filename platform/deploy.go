package platform

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/hashicorp/waypoint/builtin/docker"
)

const (
	kustomizationYAML       = "kustomization.yaml"
	basePatchDeploymentYAML = ".base-patch-deployment.yaml"
)

// https://kubectl.docs.kubernetes.io/pages/examples/kustomize.html
type DeployConfig struct {
	Namespace             string   `hcl:"namespace,optional"`
	Resources             []string `hcl:"resources,optional"`
	PatchesStrategicMerge []string `hcl:"patchesStrategicMerge,optional"`
}

type Platform struct {
	config DeployConfig
}

// Implement Configurable
func (p *Platform) Config() interface{} {
	return &p.config
}

// Implement ConfigurableNotify
func (p *Platform) ConfigSet(config interface{}) error {
	_, ok := config.(*DeployConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("Expected *DeployConfig as parameter")
	}

	// TODO(tcnksm): Write validation

	return nil
}

// Implement Builder
func (p *Platform) DeployFunc() interface{} {
	// return a function which will be called by Waypoint
	return p.deploy
}

func (b *Platform) deploy(ctx context.Context, ui terminal.UI, dockerImage *docker.Image) (*Deployment, error) {
	u := ui.Status()
	defer u.Close()
	u.Update("Deploy application")

	// Add default patch
	b.config.PatchesStrategicMerge = append(b.config.PatchesStrategicMerge, basePatchDeploymentYAML)

	kustomizationFile, err := os.Create(kustomizationYAML)
	if err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to create %s", kustomizationYAML))
		return nil, err
	}
	defer kustomizationFile.Close()

	if err := templateExec(kustomizationFile, kustomizationTmpl, b.config); err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to execute template for %s", kustomizationYAML))
		return nil, err
	}

	basePatchDeploymentFile, err := os.Create(basePatchDeploymentYAML)
	if err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to create %s", basePatchDeploymentYAML))
		return nil, err
	}
	defer basePatchDeploymentFile.Close()

	v := struct {
		ImageName string
	}{
		ImageName: dockerImage.Name(),
	}
	if err := templateExec(basePatchDeploymentFile, basePatchDeploymentTmpl, &v); err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to execute template for %s", basePatchDeploymentYAML))
		return nil, err

	}

	// (1) Create kustomization.yaml
	// (2) Build manifest
	// (3) Apply manifest to the cluster

	u.Step(terminal.StatusOK, fmt.Sprintf("deploy %s", dockerImage.Name()))

	return &Deployment{}, nil
}
