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
	preferConfigMapsHookName = "prefer-parent-configmaps-hook"

	// SkipPreferConfigMapsHook is the annotation key that, if any value is set, will cause this
	// plugin to skip preferring the parent (physical/real) configmap resources.
	SkipPreferConfigMapsHook = "skip-prefer-parent-configmaps-hook"
)

// NewPreferParentConfigmapsHook returns a PreferParentConfigmapsHook hook.ClientHook.
func NewPreferParentConfigmapsHook(ctx *syncercontext.RegisterContext) hook.ClientHook {
	return &PreferParentConfigmapsHook{
		translator: translator.NewNamespacedTranslator(
			ctx,
			"configmap",
			&corev1.ConfigMap{},
		),
		physicalNamespace: ctx.TargetNamespace,
		physicalClient:    ctx.PhysicalManager.GetClient(),
		virtualClient:     ctx.VirtualManager.GetClient(),
	}
}

// PreferParentConfigmapsHook is a hook.ClientHook implementation that will prefer configmaps from
// the physical/parent cluster over those created by/from the vcluster itself. The goal/idea here
// is that users can create a single vcluster namespace in the parent cluster, and create some
// configmaps that potentially many vcluster resources may use.
type PreferParentConfigmapsHook struct {
	translator        translator.NamespacedTranslator
	physicalNamespace string
	physicalClient    ctrlruntimeclient.Client
	virtualClient     ctrlruntimeclient.Client
}

// Name returns the name of the ClientHook.
func (h *PreferParentConfigmapsHook) Name() string {
	return preferConfigMapsHookName
}

// Resource returns the type of resource the ClientHook mutates.
func (h *PreferParentConfigmapsHook) Resource() ctrlruntimeclient.Object {
	return &corev1.Pod{}
}

var _ hook.MutateCreatePhysical = &PreferParentConfigmapsHook{}

// MutateCreatePhysical mutates incoming physical cluster create operations to determine if the pod
// being created refers to a configmap that exists in the physical cluster, if "yes", we replace
// the configmap reference of the vcluster created configmap with the "real" configmap.
func (h *PreferParentConfigmapsHook) MutateCreatePhysical(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
) (ctrlruntimeclient.Object, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("%w: object %v is not a pod", ErrWrongResourceType, obj)
	}

	skip, ok := pod.Annotations[SkipPreferConfigMapsHook]
	if ok {
		if len(skip) > 0 {
			return pod, nil
		}
	}

	configEnvs := FindMountedEnvsOfType(&pod.Spec, configMap)
	configVols := FindMountedVolumesOfType(&pod.Spec, configMap)

	fmt.Println("envs/vols->", configEnvs, configVols)

	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].VolumeSource.ConfigMap == nil {
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
		// wrong about that!
		if vPod.Spec.Volumes[i].VolumeSource.ConfigMap == nil {
			continue
		}

		translatedVolumeName := translate.PhysicalName(
			vPod.Spec.Volumes[i].VolumeSource.ConfigMap.Name,
			vPod.Namespace,
		)

		if translatedVolumeName == volume.VolumeSource.ConfigMap.Name {
			pVolumeName = vPod.Spec.Volumes[i].VolumeSource.ConfigMap.Name
		}

		// we should *not* ever hit this because we should always have alignment between the virtual
		// and physical objects
		if pVolumeName == "" {
			continue
		}

		realConfigMap := &corev1.ConfigMap{}

		err = h.physicalClient.Get(
			ctx,
			types.NamespacedName{
				Name:      pVolumeName,
				Namespace: h.physicalNamespace,
			},
			realConfigMap,
		)

		// we did not find a real configmap matching the virtual pods configmap name
		if err != nil {
			continue
		}

		volume.VolumeSource.ConfigMap.Name = pVolumeName
	}

	MutateAnnotations(pod, preferConfigMapsHookName)

	return pod, nil
}

var _ hook.MutateUpdatePhysical = &PreferParentConfigmapsHook{}

// MutateUpdatePhysical mutates incoming physical cluster update operations to make sure we are
// enforcing the plugin annotations on the physical resources.
func (h *PreferParentConfigmapsHook) MutateUpdatePhysical(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
) (ctrlruntimeclient.Object, error) {
	_ = ctx

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("%w: object %v is not a pod", ErrWrongResourceType, obj)
	}

	MutateAnnotations(pod, preferConfigMapsHookName)

	return pod, nil
}
