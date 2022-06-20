# vCluster Plugin: Prefer Parent Resources

This [vCluster](https://github.com/loft-sh/vcluster) plugin modifies the behavior of the default 
pod syncer to prefer resources from the *parent* (or physical, or host) cluster to those that 
are created in the virtual cluster.

What does that actually mean? Assume that you have created a configmap in your virtual cluster 
called "my-configmap". Later, you create a pod that mounts this configmap as a volume. When you 
deploy this pod, vcluster will automagically replicate that configmap form the vcluster into the 
parent physical cluster's namespace. In doing so the name will be translated such that there are 
no conflicts. When the pod is actually created in the parent/physical cluster, it will be 
created with the newly mapped and name translated configmap attached. 

This behavior is great, and is a core functionality of vcluster, however, there may be scenarios 
where you *don't* want to mount the configmap from within the virtual cluster. Instead, you may 
actually prefer to mount configmaps (for now, but also could be secrets and probably other 
things in the near future!) from the parent/physical cluster namespace where your virtual 
cluster resides.

The use case for this could be that you are a developer who has been given a vcluster for your 
development environment, but the cluster admin folks have already deployed the configmaps that 
you (and perhaps your other development teams in *other* vclusters) need to mount. With this 
plugin enabled, if a pod is deployed with a configmap in a volume mount, the plugin will check 
to see if that is a valid configmap name in the parent/physical cluster, and if so, that is the 
configmap that will be mounted. This behavior can be disabled by adding an annotation with a key 
of "skip-prefer-parent-configmaps-hook" (or in the future "secrets" or whatever other hooks) 
with any non-empty string value.