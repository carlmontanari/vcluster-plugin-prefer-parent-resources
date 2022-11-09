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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	preferSecretsHookName = "prefer-parent-secrets-hook"

	// SkipPreferSecretsHook is the annotation key that, if any value is set, will cause this
	// plugin to skip preferring the parent (physical/real) resources.
	SkipPreferSecretsHook = "skip-prefer-parent-secrets-hook"
)

// NewPreferParentSecretsHook returns a PreferParentSecretsHook hook.ClientHook.
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
	physicalClient    client.Client
	virtualClient     client.Client
}

// Name returns the name of the ClientHook.
func (h *PreferParentSecretsHook) Name() string {
	return preferSecretsHookName
}

// Resource returns the type of resource the ClientHook mutates.
func (h *PreferParentSecretsHook) Resource() client.Object {
	return &corev1.Pod{}
}

var _ hook.MutateCreatePhysical = &PreferParentSecretsHook{}

// MutateCreatePhysical mutates incoming physical cluster create operations to determine if the pod
// being created refers to a secret that exists in the physical cluster, if "yes", we replace
// the secret reference of the vcluster created secret with the "real" secret.
func (h *PreferParentSecretsHook) MutateCreatePhysical(
	ctx context.Context,
	obj client.Object,
) (client.Object, error) {
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

	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].VolumeSource.Secret == nil {
			continue
		}

		volume := pod.Spec.Volumes[i]

		// get the "real" name of the pod (as in "real" in the vcluster)
		vName := pod.Annotations[translator.NameAnnotation]
		vNamespace := pod.Annotations[translator.NamespaceAnnotation]

		vPod := &corev1.Pod{}

		err := h.virtualClient.Get(
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

		var pVolumeName string

		// will the volumes always be in the same order? assuming "yes" for now, but open to being
		// wrong about that!k
		if vPod.Spec.Volumes[i].VolumeSource.Secret == nil {
			continue
		}

		translatedVolumeName := translate.PhysicalName(
			vPod.Spec.Volumes[i].VolumeSource.Secret.SecretName,
			vPod.Namespace,
		)

		if translatedVolumeName == volume.VolumeSource.Secret.SecretName {
			pVolumeName = vPod.Spec.Volumes[i].VolumeSource.Secret.SecretName
		}

		// we should *not* ever hit this because we should always have alignment between the virtual
		// and physical objects
		if pVolumeName == "" {
			continue
		}

		realSecret := &corev1.Secret{}

		err = h.physicalClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pVolumeName,
				Namespace: h.physicalNamespace,
			},
			realSecret,
		)

		// we did not find a real secret matching the virtual pods secret name
		if err != nil {
			continue
		}

		volume.VolumeSource.Secret.SecretName = pVolumeName
	}

	MutateAnnotations(pod, preferSecretsHookName)

	return pod, nil
}

var _ hook.MutateUpdatePhysical = &PreferParentSecretsHook{}

// MutateUpdatePhysical mutates incoming physical cluster update operations to make sure we are
// enforcing the plugin annotations on the physical resources.
func (h *PreferParentSecretsHook) MutateUpdatePhysical(
	ctx context.Context,
	obj client.Object,
) (client.Object, error) {
	_ = ctx

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("%w: object %v is not a pod", ErrWrongResourceType, obj)
	}

	MutateAnnotations(pod, preferSecretsHookName)

	return pod, nil
}
