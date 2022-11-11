package hooks

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster-sdk/translate"
	"k8s.io/apimachinery/pkg/types"

	"github.com/loft-sh/vcluster-sdk/hook"
	syncercontext "github.com/loft-sh/vcluster-sdk/syncer/context"
	"github.com/loft-sh/vcluster-sdk/syncer/translator"
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
func NewPreferParentSecretsHook(ctx *syncercontext.RegisterContext) hook.ClientHook {
	return &PreferParentSecretsHook{
		translator: translator.NewNamespacedTranslator(
			ctx,
			"secret",
			&corev1.Secret{},
		),
		physicalNamespace: ctx.TargetNamespace,
		physicalClient:    ctx.PhysicalManager.GetClient(),
		virtualClient:     ctx.VirtualManager.GetClient(),
	}
}

// PreferParentSecretsHook is a hook.ClientHook implementation that will prefer secrets from
// the physical/parent cluster over those created by/from the vcluster itself. The goal/idea here
// is that users can create a single vcluster namespace in the parent cluster, and create some
// secrets that potentially many vcluster resources may use.
type PreferParentSecretsHook struct {
	translator        translator.NamespacedTranslator
	physicalNamespace string
	physicalClient    ctrlruntimeclient.Client
	virtualClient     ctrlruntimeclient.Client
}

// Name returns the name of the ClientHook.
func (h *PreferParentSecretsHook) Name() string {
	return preferSecretsHookName
}

// Resource returns the type of resource the ClientHook mutates.
func (h *PreferParentSecretsHook) Resource() ctrlruntimeclient.Object {
	return &corev1.Pod{}
}

var _ hook.MutateCreatePhysical = &PreferParentConfigmapsHook{}

func (h *PreferParentSecretsHook) mutateCreatePhysicalSecretEnvs(
	secretEnvs []EnvAtPos,
	ctx context.Context,
	pod, vPod *corev1.Pod,
) *corev1.Pod {
	for i := range secretEnvs {
		var pEnvRefName string

		for envI := range vPod.Spec.Containers[secretEnvs[i].containerPos].Env {
			vObjName := vPod.Spec.Containers[secretEnvs[i].containerPos].Env[envI].
				ValueFrom.SecretKeyRef.LocalObjectReference.Name

			translatedEnvRefName := translate.PhysicalName(
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

		err := h.physicalClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pEnvRefName,
				Namespace: h.physicalNamespace,
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

func (h *PreferParentSecretsHook) mutateCreatePhysicalSecretVols(
	secretVols []VolAtPos,
	ctx context.Context,
	pod, vPod *corev1.Pod,
) *corev1.Pod {
	for i := range secretVols {
		var pVolumeName string

		translatedVolumeName := translate.PhysicalName(
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

		err := h.physicalClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pVolumeName,
				Namespace: h.physicalNamespace,
			},
			realSecret,
		)
		// we did not find a real configmap matching the virtual pods configmap name
		if err != nil {
			continue
		}

		pod.Spec.Volumes[secretVols[i].pos].VolumeSource.Secret.SecretName = pVolumeName
	}

	return pod
}

// MutateCreatePhysical mutates incoming physical cluster create operations to determine if the pod
// being created refers to a secret that exists in the physical cluster, if "yes", we replace
// the secret reference of the vcluster created secret with the "real" secret.
func (h *PreferParentSecretsHook) MutateCreatePhysical(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
) (ctrlruntimeclient.Object, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("%w: object %v is not a pod", ErrWrongResourceType, obj)
	}

	skip, ok := pod.Annotations[SkipPreferSecretsHook]
	if ok {
		if len(skip) > 0 {
			return pod, nil
		}
	}

	secretEnvs := FindMountedEnvsOfType(&pod.Spec, secret)
	secretVols := FindMountedVolumesOfType(&pod.Spec, secret)

	if len(secretEnvs) == 0 && len(secretVols) == 0 {
		// nothing to do, we're outta here!
		return pod, nil
	}

	MutateAnnotations(pod, preferSecretsHookName)

	vPod, err := GetVirtualPod(ctx, pod, h.virtualClient)
	if err != nil {
		return nil, err
	}

	if len(secretEnvs) > 0 {
		pod = h.mutateCreatePhysicalSecretEnvs(secretEnvs, ctx, pod, vPod)
	}

	if len(secretVols) > 0 {
		pod = h.mutateCreatePhysicalSecretVols(secretVols, ctx, pod, vPod)
	}

	return pod, nil
}

var _ hook.MutateUpdatePhysical = &PreferParentConfigmapsHook{}

// MutateUpdatePhysical mutates incoming physical cluster update operations to make sure we are
// enforcing the plugin annotations on the physical resources.
func (h *PreferParentSecretsHook) MutateUpdatePhysical(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
) (ctrlruntimeclient.Object, error) {
	_ = ctx

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("%w: object %v is not a pod", ErrWrongResourceType, obj)
	}

	MutateAnnotations(pod, preferSecretsHookName)

	return pod, nil
}
