# Prefer "Parent" Cluster Resources(s)

The default behavior of this vcluster plugin is to prefer resources from the "parent" (or 
physical/host) cluster. This example is a basic showcase of this behavior.


# Setup

Assuming you already have a cluster setup and appropriate kubeconfig sorted, you can start by 
creating a vcluster. (the distro flag is optional, my homelab has NFS storage which k3s does not appreciate!)

`vcluster create my-vcluster -n my-vcluster -f plugin.yaml --connect=false --distro=k0s`

The above command would be run from the root of this repo, and assumes you have built this 
plugin image and updated the plugin.yaml to point the pull-able image. We set connect to false 
so that we don't set the context to the vcluster but rather just leave it set as the parent cluster.

Next we need to create some real resources (configmap and secret) in the parent cluster -- this will
be the resources that we want the vcluster pod(s) to use -- rather than resources that may or may not 
exist in the vcluster (and which would get synced into the parent cluster by the vcluster syncer).

`kubectl apply -f examples/prefer-parent-configmap-and-secret/parent-manifests/`


# Try It Out

Now that the vcluster is set up, and some "real" resources have been created in the parent cluster,
it is time to test out the plugin!

`make connect-vcluster && sleep 1 && KUBECONFIG=./kubeconfig.yaml kubectl apply -f examples/prefer-parent-configmap-and-secret/vcluster-manifests`

*Note* the make directive runs connect and sends it to the background -- this makes vcluster 
generate the kubeconfig file we can use to connect to the vcluster. You could of course just use 
vcluster with the (default) connect flag set, but we want to pop back and forth and this is an 
easy way to do that.


# Validate

With the pod deployed, we now need to check firstly if the pod is even running, and secondly, if 
it is, are the "real" resources mounted appropriately?

`kubectl get pods -n my-vcluster | grep debian`

Should show the pod from the vcluster is in fact up and running.

The pod we deployed has two containers, the first `debian-volumes` showcases replacing volumes backed
by configmaps/secrets. We can check on the volumes that are mounted on the pod with the following
command:

`kubectl get pods -n my-vcluster $(kubectl get pods -n my-vcluster | grep debian | awk '{ print $1 }') -o jsonpath='{.spec.volumes}' | jq 'del(.[] | select(.name | startswith("kube-api-access")))'`

The above command should show us four volumes mounted -- two are ones we expect to be "real", and 
two we expect to be "virtual" (from within our vcluster). The output should be similar to the
following:

```json
[
  {
    "configMap": {
      "defaultMode": 420,
      "name": "virtual-configmap-x-vcluster-x-my-vcluster"
    },
    "name": "virtual-configmap"
  },
  {
    "name": "virtual-secret",
    "secret": {
      "defaultMode": 420,
      "secretName": "virtual-secret-x-vcluster-x-my-vcluster"
    }
  },
  {
    "configMap": {
      "defaultMode": 420,
      "name": "real-configmap"
    },
    "name": "configmap"
  },
  {
    "name": "secret",
    "secret": {
      "defaultMode": 420,
      "secretName": "real-secret"
    }
  }
]
```

You can see the volume named `virtual-configmap` has a "name" in the `configMap` section that has
been "translated" by the vcluster syncer -- this means this volume is mapped to a resource in the
virtual cluster. The same can be seen on the `virtual-secret` volume. The other two volumes, named
`configmap` and `secret` have "real" resources mounted - you can see this because they do not have a
translated name!

Next we want to see if the environment variables work as well. We can see this on the second
container in the pod, so once again a little jq magic to make the output nicer to investigate:

`kubectl get pods -n my-vcluster $(kubectl get pods -n my-vcluster | grep debian | awk '{ print $1 }') -o jsonpath='{.spec.containers[1].env}' | jq 'del(.[] | select(.name | startswith("KUBERNETES")))'`

And we should get some output similar to:

```json
[
  {
    "name": "config",
    "valueFrom": {
      "configMapKeyRef": {
        "key": "redis.conf",
        "name": "real-configmap",
        "optional": false
      }
    }
  },
  {
    "name": "secret",
    "valueFrom": {
      "secretKeyRef": {
        "key": "client-id",
        "name": "real-secret",
        "optional": false
      }
    }
  },
  {
    "name": "virtual-config",
    "valueFrom": {
      "configMapKeyRef": {
        "key": "redis.conf",
        "name": "virtual-configmap-x-vcluster-x-my-vcluster",
        "optional": false
      }
    }
  },
  {
    "name": "virtual-secret",
    "valueFrom": {
      "secretKeyRef": {
        "key": "client-id",
        "name": "virtual-secret-x-vcluster-x-my-vcluster",
        "optional": false
      }
    }
  }
]
```

In the above output the `config` and `secret` envs should map to "real" resources, and sure enough
they do!

That is pretty much the entire point of the plugin! Prefer configmaps and secrets that exist in the
"real" cluster, otherwise behave like "normal"!


# Clean Up

Clean up the "real" configmap in the parent cluster:

`kubectl delete -f examples/prefer-parent-configmap-and-secret/parent-manifests/`

And nicely clean up the vcluster: 

`vcluster delete my-vcluster`
