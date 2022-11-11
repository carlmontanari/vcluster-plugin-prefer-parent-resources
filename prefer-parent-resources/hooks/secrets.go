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
	preferSecretsHookName = "prefer-parent-secrets-hook"

	// SkipPreferSecretsHook is the annotation key that, if any value is set, will cause this
	// plugin to skip preferring the parent (physical/real) secret resources.
	SkipPreferSecretsHook = "skip-prefer-parent-secrets-hook"
)

// NewPreferParentSecretsHook returns a NewPreferParentSecretsHook hook.ClientHook.
func NewPreferParentSecretsHook(ctx *vclustersdksyncercontext.RegisterContext) EnvVolMutatingHook {
	return &envVolMutatingHook{
		name:             preferSecretsHookName,
		ignoreAnnotation: SkipPreferSecretsHook,
		mutateType:       &corev1.Secret{},
		translator: vclustersdksyncertranslator.NewNamespacedTranslator(
			ctx,
			secret,
			&corev1.Secret{},
		),
		physicalNamespace: ctx.TargetNamespace,
		physicalClient:    ctx.PhysicalManager.GetClient(),
		virtualClient:     ctx.VirtualManager.GetClient(),
		envMutator:        mutateCreatePhysicalSecretEnvs,
		volMutator:        mutateCreatePhysicalSecretVols,
	}
}

// PreferParentSecretsHook is a hook.ClientHook implementation that will prefer secrets from
// the physical/parent cluster over those created by/from the vcluster itself. The goal/idea here
// is that users can create a single vcluster namespace in the parent cluster, and create some
// secrets that potentially many vcluster resources may use.
type PreferParentSecretsHook struct {
	EnvVolMutatingHook
}

func mutateCreatePhysicalSecretEnvs(
	ctx context.Context,
	physicalClient ctrlruntimeclient.Client,
	physicalNamespace string,
	secretEnvs []EnvAtPos,
	pod, vPod *corev1.Pod,
) *corev1.Pod {
	for i := range secretEnvs {
		var pEnvRefName string

		for envI := range vPod.Spec.Containers[secretEnvs[i].containerPos].Env {
			vObjName := vPod.Spec.Containers[secretEnvs[i].containerPos].Env[envI].
				ValueFrom.SecretKeyRef.LocalObjectReference.Name

			translatedEnvRefName := vclustersdktranslate.PhysicalName(
				vObjName,
				vPod.Namespace,
			)

			if translatedEnvRefName == secretEnvs[i].env.ValueFrom.SecretKeyRef.LocalObjectReference.Name {
				pEnvRefName = vObjName

				break
			}
		}

		if pEnvRefName == "" {
			continue
		}

		realSecret := &corev1.Secret{}

		err := physicalClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pEnvRefName,
				Namespace: physicalNamespace,
			},
			realSecret,
		)
		// we did not find a real configmap matching the virtual pods configmap name
		if err != nil {
			continue
		}

		pod.Spec.Containers[secretEnvs[i].containerPos].Env[secretEnvs[i].envPos].ValueFrom.
			SecretKeyRef.LocalObjectReference.Name = pEnvRefName
	}

	return pod
}

func mutateCreatePhysicalSecretVols(
	ctx context.Context,
	physicalClient ctrlruntimeclient.Client,
	physicalNamespace string,
	secretVols []VolAtPos,
	pod, vPod *corev1.Pod,
) *corev1.Pod {
	for i := range secretVols {
		var pVolumeName string

		translatedVolumeName := vclustersdktranslate.PhysicalName(
			vPod.Spec.Volumes[secretVols[i].pos].Secret.SecretName,
			vPod.Namespace,
		)

		if translatedVolumeName == secretVols[i].vol.Secret.SecretName {
			pVolumeName = vPod.Spec.Volumes[i].VolumeSource.Secret.SecretName
		}

		// we should *not* ever hit this because we should always have alignment between the virtual
		// and physical objects
		if pVolumeName == "" {
			continue
		}

		realSecret := &corev1.Secret{}

		err := physicalClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pVolumeName,
				Namespace: physicalNamespace,
			},
			realSecret,
		)
		// we did not find a real configmap matching the virtual pods configmap name
		// TODO this should check for notfound vs some other errors and behave accordingly
		if err != nil {
			continue
		}

		pod.Spec.Volumes[secretVols[i].pos].VolumeSource.Secret.SecretName = pVolumeName
	}

	return pod
}
