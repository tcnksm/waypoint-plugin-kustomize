package platform

import (
	"io"
	"text/template"
)

var kustomizationTmpl = template.Must(template.New("kustomization").Parse(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
{{ if .Namespace }}namespace: {{ .Namespace }}{{ "\n" }}{{ end -}}
{{ if .Resources }}resources:{{ "\n" }}{{ end -}}
{{ range .Resources }}- {{ . }}{{ "\n" }}{{ end -}}
{{ if .PatchesStrategicMerge }}patchesStrategicMerge:{{ "\n" }}{{ end -}}
{{ range .PatchesStrategicMerge }}- {{ . }}{{ "\n" }}{{ end -}}
`))

var basePatchDeploymentTmpl = template.Must(template.New("kustomization").Parse(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment
spec:
  template:
    spec:
      containers:
       - name: main-server
         image: {{ .ImageName }}
`))

func templateExec(w io.Writer, tmpl *template.Template, data interface{}) error {
	return tmpl.Execute(w, data)
}
