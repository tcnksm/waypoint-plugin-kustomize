package platform

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
)

func TestTemplateExec(t *testing.T) {
	cases := map[string]struct {
		tmpl *template.Template
		data interface{}
		want string
	}{
		"Kustomization": {
			tmpl: kustomizationTmpl,
			data: &DeployConfig{
				Namespace:             "example",
				Resources:             []string{"resource1.yaml", "resource2.yaml"},
				PatchesStrategicMerge: []string{"patch1.yaml", "patch2.yaml"},
			},
			want: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: example
resources:
- resource1.yaml
- resource2.yaml
patchesStrategicMerge:
- patch1.yaml
- patch2.yaml
`,
		},
		"PatchDeployment": {
			tmpl: basePatchDeploymentTmpl,
			data: &struct {
				ImageName string
			}{
				ImageName: "example-nodejs:1.0.0",
			},
			want: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment
spec:
  template:
    spec:
      containers:
       - name: main-server
         image: example-nodejs:1.0.0
`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var got bytes.Buffer
			if err := templateExec(&got, tc.tmpl, tc.data); err != nil {
				t.Fatalf("err: %s", err)
			}
			if diff := cmp.Diff(got.String(), tc.want); diff != "" {
				t.Fatalf("templateExec() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
