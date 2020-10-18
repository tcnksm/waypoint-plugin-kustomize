# Waypoint Plugin Kustomize

`waypoint-plugin-kustomize` is an experimental implementation of a platform plugin for [Waypoint](https://github.com/hashicorp/waypoint). This plugin allows you to patch kubernetes manifest by [`kustomize`](https://github.com/kubernetes-sigs/kustomize). By default, it uses [default remote base](/kustomize/remote-base/default/) which can be used for most of the applications but you can prepare your own (e.g., machine learning application-specific base). In addition to that, it automatically adds the Waypoint specific patches like image name which is built-in `build` step or [entrypoint](https://www.waypointproject.io/docs/entrypoint) environmental vars.

## Configure

The following is the basic configuration of using this plugin.

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

By default, it uses [default base](/kustomize/remote-base/default/) and adds patches of Waypoint related configuration. If you want to change the base you can change it by specifying it in the `resources` field. In `patches_strategic_merge` field, you can specify your own patches (see [example](/example/patch-deployment.ya\
 ml)).

The followings are the current limitations of this plugin:

- You must name `Deployment` resource `"deployment"`
- You must name the main application container name `"main-server"`

## Install

To install the plugin, run the following command:

```bash
$ make install
```

With this, the plugin binary is installed in `${HOME}/.config/waypoint/plugins/`.
