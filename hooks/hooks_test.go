package hooks_test

import corev1 "k8s.io/api/core/v1"

type comparePodTestCase struct {
	description string
	inPod       *corev1.Pod
	expected    *corev1.Pod
}
