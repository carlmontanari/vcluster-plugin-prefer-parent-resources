//nolint:dupl
package hooks

import (
	"context"

	vclustersdklog "github.com/loft-sh/vcluster-sdk/log"

	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"

	vclustersdktranslate "github.com/loft-sh/vcluster-sdk/translate"
	"k8s.io/apimachinery/pkg/types"

	vclustersdksyncercontext "github.com/loft-sh/vcluster-sdk/syncer/context"
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
	return newEnvVolMutatingHook(
		ctx,
		preferSecretsHookName,
		SkipPreferSecretsHook,
		&corev1.Secret{},
		mutateCreatePhysicalSecretEnvs,
		mutateCreatePhysicalSecretVols,
	)
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
	log vclustersdklog.Logger,
	physicalClient ctrlruntimeclient.Client,
	physicalNamespace string,
	secretEnvs []EnvAtPos,
	pod, vPod *corev1.Pod,
) *corev1.Pod {
	for i := range secretEnvs {
		var pEnvRefName string

		for envI := range vPod.Spec.Containers[secretEnvs[i].containerPos].Env {
			env := vPod.Spec.Containers[secretEnvs[i].containerPos].Env[envI]

			if env.ValueFrom.SecretKeyRef == nil {
				// not a secret, skip
				continue
			}

			vObjName := env.ValueFrom.SecretKeyRef.LocalObjectReference.Name

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
		if err != nil {
			// we hit some error other than not found; log and move on. otherwise we just assume
			// not found and we also move on.
			if !apimachineryerrors.IsNotFound(err) {
				log.Errorf(
					"error fetching host cluster secret '%s/%s', error: '%s', skipping...",
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
			secretEnvs[i].containerPos,
			secretEnvs[i].env.Name,
			realSecret.Namespace,
			realSecret.Name,
		)

		var replaced bool

		for envI, env := range pod.Spec.Containers[secretEnvs[i].containerPos].Env {
			if env.Name == secretEnvs[i].env.Name {
				pod.Spec.Containers[secretEnvs[i].containerPos].Env[envI].ValueFrom.
					SecretKeyRef.LocalObjectReference.Name = pEnvRefName

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
				secretEnvs[i].containerPos,
				secretEnvs[i].env.Name,
				realSecret.Namespace,
				realSecret.Name,
			)
		}
	}

	return pod
}

func mutateCreatePhysicalSecretVols(
	ctx context.Context,
	log vclustersdklog.Logger,
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
			pVolumeName = vPod.Spec.Volumes[secretVols[i].pos].VolumeSource.Secret.SecretName
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
		if err != nil {
			// we hit some error other than not found; log and move on. otherwise we just assume
			// not found and we also move on.
			if !apimachineryerrors.IsNotFound(err) {
				log.Errorf(
					"error fetching host cluster secret '%s/%s', error: '%s', skipping...",
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
			secretVols[i].pos,
			realSecret.Namespace,
			realSecret.Name,
		)

		pod.Spec.Volumes[secretVols[i].pos].VolumeSource.Secret.SecretName = pVolumeName
	}

	return pod
}
