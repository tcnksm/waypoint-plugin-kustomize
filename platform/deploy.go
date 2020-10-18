package platform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/hashicorp/waypoint/builtin/docker"
	"github.com/hashicorp/waypoint/builtin/k8s"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/kustomize/api/types"
)

const (
	defaultRemoteBase = "github.com/tcnksm/waypoint-plugin-kustomize/kustomize/remote-base/default?ref=main"

	kustomizationYAML       = "kustomization.yaml"
	basePatchDeploymentYAML = ".patch-deployment.yaml"

	outputDir = ".kustomization"
)

const (
	// https://github.com/hashicorp/waypoint/blob/main/builtin/k8s/platform.go
	labelId    = "waypoint.hashicorp.com/id"
	labelNonce = "waypoint.hashicorp.com/nonce"
)

var (
	outputYAML = filepath.Join(outputDir, "output.yaml")
)

// https://kubectl.docs.kubernetes.io/pages/examples/kustomize.html
type DeployConfig struct {
	Namespace             string            `hcl:"namespace,optional"`
	Resources             []string          `hcl:"resources,optional"`
	CommonLabels          map[string]string `hcl:"common_labels,optional"`
	PatchesStrategicMerge []string          `hcl:"patches_strategic_merge,optional"`
}

type Platform struct {
	config DeployConfig
}

// Implement Configurable
func (p *Platform) Config() (interface{}, error) {
	return &p.config, nil
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

func (b *Platform) deploy(ctx context.Context, ui terminal.UI, src *component.Source, dockerImage *docker.Image, deployConfig *component.DeploymentConfig) (*k8s.Deployment, error) {
	u := ui.Status()
	defer u.Close()

	id, err := component.Id()
	if err != nil {
		return nil, err
	}
	name := strings.ToLower(fmt.Sprintf("%s-%s", src.App, id))

	if b.config.CommonLabels == nil {
		b.config.CommonLabels = make(map[string]string)
	}
	b.config.CommonLabels["name"] = name

	if len(b.config.Resources) == 0 {
		b.config.Resources = append(b.config.Resources, defaultRemoteBase)
	}
	b.config.PatchesStrategicMerge = append(b.config.PatchesStrategicMerge, basePatchDeploymentYAML)

	sg := ui.StepGroup()
	step := sg.Add(fmt.Sprintf("Generating %s ...", kustomizationYAML))
	defer step.Abort()

	// Generate kustomization.yaml
	patchesStrategicMerge := make([]types.PatchStrategicMerge, 0, len(b.config.PatchesStrategicMerge))
	for _, path := range b.config.PatchesStrategicMerge {
		patchesStrategicMerge = append(patchesStrategicMerge, types.PatchStrategicMerge(path))
	}

	kustomization := &types.Kustomization{
		TypeMeta: types.TypeMeta{
			Kind:       types.KustomizationKind,
			APIVersion: types.KustomizationVersion,
		},
		NameSuffix:            fmt.Sprintf("-%s", name),
		Namespace:             b.config.Namespace,
		Resources:             b.config.Resources,
		CommonLabels:          b.config.CommonLabels,
		PatchesStrategicMerge: patchesStrategicMerge,
	}

	kustomizationFile, err := os.Create(kustomizationYAML)
	if err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to create %s", kustomizationYAML))
		return nil, err
	}
	defer kustomizationFile.Close()

	if err := yaml.NewEncoder(kustomizationFile).Encode(kustomization); err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to encode Kustomization to %s", kustomizationYAML))
		return nil, err
	}
	defer os.Remove(kustomizationYAML)
	step.Update(fmt.Sprintf("Successfully generated %s", kustomizationYAML))
	step.Done()

	step = sg.Add(fmt.Sprintf("Generating %s ...", basePatchDeploymentYAML))

	// https://www.waypointproject.io/docs/entrypoint
	env := []corev1.EnvVar{}
	for k, v := range deployConfig.Env() {
		env = append(env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	basePatchDeployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			// NOTE(tcnksm): This and remote-base name must be the same.
			Name: "deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						labelId: id,
					},
					Annotations: map[string]string{
						labelNonce: time.Now().UTC().Format(time.RFC3339Nano),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "main-server",
							Image: dockerImage.Name(),
							Env:   env,
						},
					},
				},
			},
		},
	}

	basePatchDeploymentFile, err := os.Create(basePatchDeploymentYAML)
	if err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to create %s", basePatchDeploymentYAML))
		return nil, err
	}
	defer basePatchDeploymentFile.Close()

	encoder := json.NewSerializerWithOptions(
		json.DefaultMetaFactory, nil, nil,
		json.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)

	if err := encoder.Encode(basePatchDeployment, basePatchDeploymentFile); err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to encode Deployment to %s", basePatchDeploymentYAML))
		return nil, err
	}
	defer os.Remove(basePatchDeploymentYAML)
	step.Update(fmt.Sprintf("Successfully generated %s", basePatchDeploymentYAML))
	step.Done()

	step = sg.Add("Generating manifest by kustomize build...")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.Mkdir(outputDir, os.ModePerm)
		if os.IsExist(err) {
			// Skip
		} else if err != nil {
			return nil, err
		}
	}

	outputFile, err := os.Create(outputYAML)
	if err != nil {
		u.Step(terminal.StatusError, fmt.Sprintf("Failed to create %s", outputYAML))
		return nil, err
	}
	defer outputFile.Close()

	var kustomizeBuildStderr bytes.Buffer
	cmdKustmizeBuild := exec.CommandContext(ctx,
		"kustomize",
		"build",
	)
	cmdKustmizeBuild.Stdout = outputFile
	cmdKustmizeBuild.Stderr = &kustomizeBuildStderr

	if err := cmdKustmizeBuild.Run(); err != nil {
		u.Step(terminal.StatusError, "Failed to run kustomize build")
		u.Step(terminal.StatusError, kustomizeBuildStderr.String())
		return nil, err
	}
	step.Update(fmt.Sprintf("Successfully finished kustomize build (see %s)", outputYAML))
	step.Done()

	step = sg.Add("Applying manifest by kubectl...")
	var (
		kubectlApplyStdout bytes.Buffer
		kubectlApplyStderr bytes.Buffer
	)
	cmdKubectlApply := exec.CommandContext(ctx,
		"kubectl",
		"apply",
		"-f",
		outputYAML,
	)
	cmdKubectlApply.Stdout = &kubectlApplyStdout
	cmdKubectlApply.Stderr = &kubectlApplyStderr

	if err := cmdKubectlApply.Run(); err != nil {
		u.Step(terminal.StatusError, "Failed to run kubectl apply")
		u.Step(terminal.StatusError, kubectlApplyStderr.String())
		return nil, err
	}
	step.Update("Successfully finished kubectl apply")
	step.Done()

	return &k8s.Deployment{
		Id:   id,
		Name: name,
	}, nil
}
