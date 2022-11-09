package hooks_test

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

type comparePodTestCase struct {
	description string
	inPod       *corev1.Pod
	expected    *corev1.Pod
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	return scheme
}
