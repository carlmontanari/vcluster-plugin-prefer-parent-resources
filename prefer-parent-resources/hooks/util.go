package hooks

import (
	"context"
	"fmt"

	vclustersdksyncertranslator "github.com/loft-sh/vcluster-sdk/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configMap = "configmap"
	secret    = "secret"
)

// MutateAnnotations ensures that the provided hook name is set for the 'mutated-by-hook'
// annotation.
func MutateAnnotations(pod *corev1.Pod, hookName string) {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	existing, ok := pod.Annotations["vcluster.loft.sh/mutated-by-hook"]
	if !ok {
		pod.Annotations["vcluster.loft.sh/mutated-by-hook"] = hookName
	} else {
		pod.Annotations["vcluster.loft.sh/mutated-by-hook"] = existing + "," + hookName
	}
}

// EnvAtPos is a simple object representing a corev1.EnvVar and its position in the container
// list, and position in the env list of that container.
type EnvAtPos struct {
	containerPos int
	envPos       int
	env          corev1.EnvVar
}

// FindMountedEnvsOfType finds all secrets and configmaps that are mounted as environment variables
// in the given pod.
func FindMountedEnvsOfType(podSpec *corev1.PodSpec, t string) []EnvAtPos {
	var envsOfType []EnvAtPos

	for containerI := range podSpec.Containers {
		for envI := range podSpec.Containers[containerI].Env {
			if podSpec.Containers[containerI].Env[envI].ValueFrom == nil {
				continue
			}

			switch t {
			case configMap:
				if podSpec.Containers[containerI].Env[envI].ValueFrom.ConfigMapKeyRef != nil {
					envsOfType = append(
						envsOfType,
						EnvAtPos{
							containerI,
							envI,
							podSpec.Containers[containerI].Env[envI],
						},
					)
				}
			case secret:
				if podSpec.Containers[containerI].Env[envI].ValueFrom.SecretKeyRef != nil {
					envsOfType = append(
						envsOfType,
						EnvAtPos{
							containerI,
							envI,
							podSpec.Containers[containerI].Env[envI],
						},
					)
				}
			}
		}
	}

	return envsOfType
}

// VolAtPos is a simple object representing the volume and its position in the volumes slice.
type VolAtPos struct {
	pos int
	vol corev1.VolumeSource
}

// FindMountedVolumesOfType finds all secrets and configmaps that are mounted as volumes in the
// given pod.
func FindMountedVolumesOfType(podSpec *corev1.PodSpec, t string) []VolAtPos {
	var volumesOfType []VolAtPos

	for i := range podSpec.Volumes {
		switch t {
		case configMap:
			if podSpec.Volumes[i].VolumeSource.ConfigMap != nil {
				volumesOfType = append(volumesOfType, VolAtPos{i, podSpec.Volumes[i].VolumeSource})
			}
		case secret:
			if podSpec.Volumes[i].VolumeSource.Secret != nil {
				volumesOfType = append(volumesOfType, VolAtPos{i, podSpec.Volumes[i].VolumeSource})
			}
		}
	}

	return volumesOfType
}

// GetVirtualPod returns the pod in the virtualClient matching the provided pod.
func GetVirtualPod(
	ctx context.Context,
	pod *corev1.Pod,
	virtualClient ctrlruntimeclient.Client,
) (*corev1.Pod, error) {
	// get the "real" name of the pod (as in "real" in the vcluster)
	vName := pod.Annotations[vclustersdksyncertranslator.NameAnnotation]
	vNamespace := pod.Annotations[vclustersdksyncertranslator.NamespaceAnnotation]

	vPod := &corev1.Pod{}

	err := virtualClient.Get(
		ctx,
		types.NamespacedName{Name: vName, Namespace: vNamespace},
		vPod,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed getting vcluster pod resource for object %s",
			ErrCantGetResource,
			pod.Name,
		)
	}

	return vPod, nil
}
