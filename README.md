# Waypoint Plugin Kustomize

```hcl
project = "example-nodejs"

app "example-nodejs" {
  deploy { 
    use "kustomize" {
      namespace = "example"
      resources = [
        "github.com/tcnksm/waypoint-plugin-kustomize/kustomize/remote-base/default?ref=main",
      ]
      patchesStrategicMerge = [
        "patch-deployment.yaml",
      ]
    }
  }
}

```
