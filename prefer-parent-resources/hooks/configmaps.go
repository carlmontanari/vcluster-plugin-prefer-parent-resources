//nolint:dupl
package hooks

import (
	"context"

	vclustersdktranslate "github.com/loft-sh/vcluster-sdk/translate"
	"k8s.io/apimachinery/pkg/types"

	vclustersdksyncercontext "github.com/loft-sh/vcluster-sdk/syncer/context"
	vclustersdksyncertranslator "github.com/loft-sh/vcluster-sdk/syncer/translator"
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
	return &envVolMutatingHook{
		name:             preferConfigMapsHookName,
		ignoreAnnotation: SkipPreferConfigMapsHook,
		mutateType:       &corev1.ConfigMap{},
		translator: vclustersdksyncertranslator.NewNamespacedTranslator(
			ctx,
			configMap,
			&corev1.ConfigMap{},
		),
		physicalNamespace: ctx.TargetNamespace,
		physicalClient:    ctx.PhysicalManager.GetClient(),
		virtualClient:     ctx.VirtualManager.GetClient(),
		envMutator:        mutateCreatePhysicalConfigMapEnvs,
		volMutator:        mutateCreatePhysicalConfigMapVols,
	}
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
	physicalClient ctrlruntimeclient.Client,
	physicalNamespace string,
	configmapEnvs []EnvAtPos,
	pod, vPod *corev1.Pod,
) *corev1.Pod {
	for i := range configmapEnvs {
		var pEnvRefName string

		for envI := range vPod.Spec.Containers[configmapEnvs[i].containerPos].Env {
			vObjName := vPod.Spec.Containers[configmapEnvs[i].containerPos].Env[envI].
				ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name

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
		// we did not find a real configmap matching the virtual pods configmap name
		if err != nil {
			continue
		}

		pod.Spec.Containers[configmapEnvs[i].containerPos].Env[configmapEnvs[i].envPos].ValueFrom.
			ConfigMapKeyRef.LocalObjectReference.Name = pEnvRefName
	}

	return pod
}

func mutateCreatePhysicalConfigMapVols(
	ctx context.Context,
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
			pVolumeName = vPod.Spec.Volumes[i].VolumeSource.ConfigMap.Name
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
		// we did not find a real configmap matching the virtual pods configmap name
		if err != nil {
			continue
		}

		pod.Spec.Volumes[configmapVols[i].pos].VolumeSource.ConfigMap.Name = pVolumeName
	}

	return pod
}
