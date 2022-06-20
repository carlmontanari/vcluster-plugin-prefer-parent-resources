package hooks

import corev1 "k8s.io/api/core/v1"

// MutateAnnotations ensures that the provided hook name is set for the 'mutated-by-hook'
// annotation.
func MutateAnnotations(pod *corev1.Pod, hookName string) {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	pod.Annotations["vcluster.loft.sh/mutated-by-hook"] = hookName
}
