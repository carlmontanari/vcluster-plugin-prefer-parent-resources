package hooks

import (
	"context"
	"fmt"

	vclustersdkhook "github.com/loft-sh/vcluster-sdk/hook"
	vclustersdklog "github.com/loft-sh/vcluster-sdk/log"
	vclustersdksyncercontext "github.com/loft-sh/vcluster-sdk/syncer/context"
	vclustersdksyncertranslator "github.com/loft-sh/vcluster-sdk/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type envMutatorFunc func(
	ctx context.Context,
	log vclustersdklog.Logger,
	physicalClient ctrlruntimeclient.Client,
	physicalNamespace string,
	atPos []EnvAtPos,
	pod, vPod *corev1.Pod,
) *corev1.Pod

type volMutatorFunc func(
	ctx context.Context,
	log vclustersdklog.Logger,
	physicalClient ctrlruntimeclient.Client,
	physicalNamespace string,
	atPos []VolAtPos, pod, vPod *corev1.Pod,
) *corev1.Pod

func newEnvVolMutatingHook(
	ctx *vclustersdksyncercontext.RegisterContext,
	name, ignoreAnnotation string,
	mutateType ctrlruntimeclient.Object,
	envMutator envMutatorFunc,
	volMutator volMutatorFunc,
) EnvVolMutatingHook {
	log := vclustersdklog.New(name)

	log.Infof("creating new hook %s", name)

	h := &envVolMutatingHook{
		ctx:               ctx,
		log:               log,
		name:              name,
		ignoreAnnotation:  ignoreAnnotation,
		mutateType:        mutateType,
		physicalNamespace: ctx.TargetNamespace,
		physicalClient:    ctx.PhysicalManager.GetClient(),
		virtualClient:     ctx.VirtualManager.GetClient(),
		envMutator:        envMutator,
		volMutator:        volMutator,
	}

	h.translator = vclustersdksyncertranslator.NewNamespacedTranslator(
		ctx,
		h.mutateTypeName(),
		mutateType,
	)

	return h
}

// EnvVolMutatingHook is an interface representing a mutating hook that operates against corev1.Pod
// objects. The concrete type should either modify configmaps or secrets mounted as volumes or as
// environment variables. This interface should probably not be implemented by any types outside
// this package and only exists for consolidating the configmap and secret mutators to avoid
// duplication.
type EnvVolMutatingHook interface {
	vclustersdkhook.ClientHook
	vclustersdkhook.MutateCreatePhysical
	vclustersdkhook.MutateUpdatePhysical
}

type envVolMutatingHook struct {
	ctx               *vclustersdksyncercontext.RegisterContext
	log               vclustersdklog.Logger
	name              string
	ignoreAnnotation  string
	mutateType        ctrlruntimeclient.Object
	translator        vclustersdksyncertranslator.NamespacedTranslator
	physicalNamespace string
	physicalClient    ctrlruntimeclient.Client
	virtualClient     ctrlruntimeclient.Client
	envMutator        envMutatorFunc
	volMutator        volMutatorFunc
}

// Name returns the name of the ClientHook.
func (h *envVolMutatingHook) Name() string {
	return h.name
}

// Resource returns the type of resource the ClientHook mutates.
func (h *envVolMutatingHook) Resource() ctrlruntimeclient.Object {
	return &corev1.Pod{}
}

// mutateTypeName returns a string representation of the envVolMutatingHook objects mutateType, that
// is, the type of resource (on a pod) that we are looking to mutate.
func (h *envVolMutatingHook) mutateTypeName() string {
	switch h.mutateType.(type) {
	case *corev1.ConfigMap:
		h.log.Debugf("creating ConfigMap translator")

		return configMap
	case *corev1.Secret:
		h.log.Debugf("creating Secret translator")

		return secret
	default:
		h.log.Errorf(
			"unknown/invalid mutate type %s, cannot create translator, panicking...", h.mutateType,
		)

		panic("unknown mutate type")
	}
}

// MutateCreatePhysical mutates incoming physical cluster create operations to determine if the pod
// being created refers to a secret or configmap that exists in the physical cluster, if "yes", we
// replace the secret or configmap reference of the vcluster created secret with the "real" object.
func (h *envVolMutatingHook) MutateCreatePhysical(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
) (ctrlruntimeclient.Object, error) {
	h.log.Debugf("mutate create physical requested")

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		h.log.Errorf("mutate create physical object is not a pod")

		return nil, fmt.Errorf("%w: object %v is not a pod", ErrWrongResourceType, obj)
	}

	h.log.Infof("mutate create physical pod %s/%s", pod.Namespace, pod.Name)

	skip, skipOk := pod.Annotations[h.ignoreAnnotation]
	if skipOk && len(skip) > 0 {
		h.log.Infof(
			"mutate create physical pod %s/%s skipping, ignore annotation set",
			pod.Namespace,
			pod.Name,
		)

		return pod, nil
	}

	envs := FindMountedEnvsOfType(&pod.Spec, h.mutateTypeName())
	vols := FindMountedVolumesOfType(&pod.Spec, h.mutateTypeName())

	if len(envs) == 0 && len(vols) == 0 {
		// nothing to do, we're outta here!
		h.log.Infof(
			"mutate create physical pod %s/%s skipping, no envvars or volumes mounted",
			pod.Namespace,
			pod.Name,
		)

		return pod, nil
	}

	MutateAnnotations(pod, h.name)

	vPod, err := GetVirtualPod(ctx, pod, h.virtualClient)
	if err != nil {
		h.log.Errorf("mutate create physical failed fetching virtual pod")

		return nil, err
	}

	if len(envs) > 0 {
		h.log.Debugf("mutate create physical mutating envs")

		pod = h.envMutator(ctx, h.log, h.physicalClient, h.physicalNamespace, envs, pod, vPod)
	}

	if len(vols) > 0 {
		h.log.Debugf("mutate create physical mutating vols")

		pod = h.volMutator(ctx, h.log, h.physicalClient, h.physicalNamespace, vols, pod, vPod)
	}

	return pod, nil
}

// MutateUpdatePhysical mutates incoming physical cluster update operations to make sure we are
// enforcing the plugin annotations on the physical resources.
func (h *envVolMutatingHook) MutateUpdatePhysical(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
) (ctrlruntimeclient.Object, error) {
	h.log.Debugf("mutate update physical requested")

	_ = ctx

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		h.log.Errorf("mutate create physical object is not a pod")

		return nil, fmt.Errorf("%w: object %v is not a pod", ErrWrongResourceType, obj)
	}

	MutateAnnotations(pod, h.name)

	return pod, nil
}
