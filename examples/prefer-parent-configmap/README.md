# Prefer "Parent" Cluster Resources(s)

The default behavior of this vcluster plugin is to prefer resources from the "parent" (or 
physical/host) cluster. This example is a basic showcase of this behavior.


# Setup

Assuming you already have a cluster setup and appropriate kubeconfig sorted, you can start by 
creating a vcluster.

`vcluster create my-vcluster -n my-vcluster -f plugin.yaml --connect=false`

The above command would be run from the root of this repo, and assumes you have built this 
plugin image and updated the plugin.yaml to point the pull-able image. We set connect to false 
so that we don't set the context to the vcluster but rather just leave it set as the parent cluster.

Next we need to create a configmap in the parent cluster -- this will be the configmap that we 
want the vcluster pod(s) to use -- rather than a configmap that may or may not exist in the 
vcluster (and which would get synced into the parent cluster by the vcluster syncer).

`kubectl apply -f examples/prefer-parent-configmap/parent-manifests/configmap.yaml`


# Try It Out

Now that the vcluster is set up, and a "real" configmap is created in the parent cluster, it is 
time to test out the plugin!

`make connect-vcluster && sleep 1 && KUBECONFIG=./kubeconfig.yaml kubectl apply -f examples/prefer-parent-configmap/vcluster-manifests`

*Note* the make directive runs connect and sends it to the background -- this makes vcluster 
generate the kubeconfig file we can use to connect to the vcluster. You could of course just use 
vcluster with the (default) connect flag set, but we want to pop back and forth and this is an 
easy way to do that.


# Validate

With the pod deployed, we now need to check firstly if the pod is even running, and secondly, if 
it is, is the "real" configmap mounted to the pod?

`kubectl get pods -n my-vcluster | grep nginx`

Should show the pod from the vcluster is in fact up and running.

`kubectl get pods -n my-vcluster $(kubectl get pods -n my-vcluster | grep nginx | awk '{ print $1 }') -o jsonpath='{.spec.volumes[0].configMap.name}'`

Should show that the name of the mounted volume is "real-configmap", which is of course the 
"real" configmap in the parent cluster. In "normal" vcluster operations, if we had created a 
configmap in the vcluster and mounted it to the pod we would see the translated name of the 
configmap in the pod output (something like real-configmap-x-vcluster-x-myv-vcluster). In fact, 
because this plugin simply uses mutate hooks to modify the standard vcluster syncer behavior, 
the configmap that was deployed in the vcluster (also named real-configmap) does in fact still 
get created in the parent cluster despite it being unused in the physical/parent cluster. If 
there was never a "real-configmap" created in the vcluster, you would not see this translated 
configmap get created.


# Clean Up

Clean up the "real" configmap in the parent cluster:

`kubectl delete -f examples/prefer-parent-configmap/parent-manifests/configmap.yaml`

And nicely clean up the vcluster: 

`vcluster delete my-vcluster`
