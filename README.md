# Waypoint Plugin Kustomize

```hcl
project = "example-nodejs"

app "example-nodejs" {
  deploy { 
    use "kustomize" {
      namespace = "example"
      resources = [
        "base-deployment.yaml",
      ]
      patchesStrategicMerge = [
        "patch-deployment.yaml",
      ]
    }
  }
}

```
