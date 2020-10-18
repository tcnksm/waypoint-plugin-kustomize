# Waypoint Plugin Kustomize

## Configure

```hcl
project = "example-nodejs"

app "example-nodejs" {
  deploy { 
    use "kustomize" {
      namespace = "example"
      patchesStrategicMerge = [
        "patch-deployment.yaml",
      ]
    }
  }
}
```

## Install

To install the plugin, run the following command:

```bash
$ make build && make install
```

With this, the plugin binary is installed in `${HOME}/.config/waypoint/plugins/`.
