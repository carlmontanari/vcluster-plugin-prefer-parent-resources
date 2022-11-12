//nolint:dupl
package hooks

import (
	"context"

	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"

	vclustersdklog "github.com/loft-sh/vcluster-sdk/log"

	vclustersdktranslate "github.com/loft-sh/vcluster-sdk/translate"
	"k8s.io/apimachinery/pkg/types"

	vclustersdksyncercontext "github.com/loft-sh/vcluster-sdk/syncer/context"
	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	preferConfigMapsHookName = "prefer-parent-configmaps-hook"

	// SkipPreferConfigMapsHook is the annotation key that, if any value is set, will cause this
	// plugin to skip preferring the parent (physical/real) configmap resources.
	SkipPreferConfigMapsHook = "skip-prefer-parent-configmaps-hook"
)

// NewPreferParentConfigmapsHook returns a PreferParentConfigmapsHook hook.ClientHook.
func NewPreferParentConfigmapsHook(
	ctx *vclustersdksyncercontext.RegisterContext,
) EnvVolMutatingHook {
	return newEnvVolMutatingHook(
		ctx,
		preferConfigMapsHookName,
		SkipPreferConfigMapsHook,
		&corev1.ConfigMap{},
		mutateCreatePhysicalConfigMapEnvs,
		mutateCreatePhysicalConfigMapVols,
	)
}

// PreferParentConfigmapsHook is a hook.ClientHook implementation that will prefer configmaps from
// the physical/parent cluster over those created by/from the vcluster itself. The goal/idea here
// is that users can create a single vcluster namespace in the parent cluster, and create some
// configmaps that potentially many vcluster resources may use.
type PreferParentConfigmapsHook struct {
	EnvVolMutatingHook
}

func mutateCreatePhysicalConfigMapEnvs(
	ctx context.Context,
	log vclustersdklog.Logger,
	physicalClient ctrlruntimeclient.Client,
	physicalNamespace string,
	configmapEnvs []EnvAtPos,
	pod, vPod *corev1.Pod,
) *corev1.Pod {
	for i := range configmapEnvs {
		var pEnvRefName string

		for envI := range vPod.Spec.Containers[configmapEnvs[i].containerPos].Env {
			env := vPod.Spec.Containers[configmapEnvs[i].containerPos].Env[envI]

			if env.ValueFrom.ConfigMapKeyRef == nil {
				// not a configmap, skip
				continue
			}

			vObjName := env.ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name

			translatedEnvRefName := vclustersdktranslate.PhysicalName(
				vObjName,
				vPod.Namespace,
			)

			if translatedEnvRefName == configmapEnvs[i].env.ValueFrom.ConfigMapKeyRef.
				LocalObjectReference.Name {
				pEnvRefName = vObjName

				break
			}
		}

		if pEnvRefName == "" {
			continue
		}

		realConfigMap := &corev1.ConfigMap{}

		err := physicalClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pEnvRefName,
				Namespace: physicalNamespace,
			},
			realConfigMap,
		)
		if err != nil {
			// we hit some error other than not found; log and move on. otherwise we just assume
			// not found and we also move on.
			if !apimachineryerrors.IsNotFound(err) {
				log.Errorf(
					"error fetching host cluster configmap '%s/%s', error: '%s', skipping...",
					physicalNamespace,
					pEnvRefName,
					err,
				)
			}

			continue
		}

		log.Infof(
			"mutating pod '%s/%s' container at index '%d', env name '%s' to mount real volume '%s/%s'",
			pod.Namespace,
			pod.Name,
			configmapEnvs[i].containerPos,
			configmapEnvs[i].env.Name,
			realConfigMap.Namespace,
			realConfigMap.Name,
		)

		var replaced bool

		for envI, env := range pod.Spec.Containers[configmapEnvs[i].containerPos].Env {
			if env.Name == configmapEnvs[i].env.Name {
				pod.Spec.Containers[configmapEnvs[i].containerPos].Env[envI].ValueFrom.
					ConfigMapKeyRef.LocalObjectReference.Name = pEnvRefName

				replaced = true

				break
			}
		}

		if !replaced {
			log.Errorf(
				"failed mutating pod '%s/%s' container at index '%d', env name '%s' "+
					"to mount real volume '%s/%s'",
				pod.Namespace,
				pod.Name,
				configmapEnvs[i].containerPos,
				configmapEnvs[i].env.Name,
				realConfigMap.Namespace,
				realConfigMap.Name,
			)
		}
	}

	return pod
}

func mutateCreatePhysicalConfigMapVols(
	ctx context.Context,
	log vclustersdklog.Logger,
	physicalClient ctrlruntimeclient.Client,
	physicalNamespace string,
	configmapVols []VolAtPos,
	pod, vPod *corev1.Pod,
) *corev1.Pod {
	for i := range configmapVols {
		var pVolumeName string

		translatedVolumeName := vclustersdktranslate.PhysicalName(
			vPod.Spec.Volumes[configmapVols[i].pos].ConfigMap.Name,
			vPod.Namespace,
		)

		if translatedVolumeName == configmapVols[i].vol.ConfigMap.Name {
			pVolumeName = vPod.Spec.Volumes[configmapVols[i].pos].VolumeSource.ConfigMap.Name
		}

		// we should *not* ever hit this because we should always have alignment between the virtual
		// and physical objects
		if pVolumeName == "" {
			continue
		}

		realConfigMap := &corev1.ConfigMap{}

		err := physicalClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pVolumeName,
				Namespace: physicalNamespace,
			},
			realConfigMap,
		)
		if err != nil {
			// we hit some error other than not found; log and move on. otherwise we just assume
			// not found and we also move on.
			if !apimachineryerrors.IsNotFound(err) {
				log.Errorf(
					"error fetching host cluster configmap '%s/%s', error: '%s', skipping...",
					physicalNamespace,
					pVolumeName,
					err,
				)
			}

			continue
		}

		log.Infof(
			"mutating pod '%s/%s' volume at index '%d' to mount real volume '%s/%s'",
			pod.Namespace,
			pod.Name,
			configmapVols[i].pos,
			realConfigMap.Namespace,
			realConfigMap.Name,
		)

		pod.Spec.Volumes[configmapVols[i].pos].VolumeSource.ConfigMap.Name = pVolumeName
	}

	return pod
}
