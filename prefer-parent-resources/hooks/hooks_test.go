package hooks_test

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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

func falsePtr() *bool {
	f := false

	return &f
}

type testPreferParentEnvVolTestCase struct {
	description  string
	pClientObjs  []runtime.Object
	vClientObjs  []runtime.Object
	mutateObj    ctrlruntimeclient.Object
	volPos       int
	containerPos int
	envPos       int
	expected     string
}
