# Waypoint Plugin Kustomize

## Configure

```hcl
project = "example-nodejs"

app "example-nodejs" {
  deploy { 
    use "kustomize" {
      patches_strategic_merge = [
        "patch-deployment.yaml",
      ]
    }
  }
}
```

## Install

To install the plugin, run the following command:

```bash
$ make install
```

With this, the plugin binary is installed in `${HOME}/.config/waypoint/plugins/`.
