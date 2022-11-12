package hooks_test

import (
	"context"
	"testing"

	vclustersdksyncercontext "github.com/loft-sh/vcluster-sdk/syncer/context"

	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/prefer-parent-resources/hooks"
	"github.com/google/go-cmp/cmp"
	vclustersdksyncertesting "github.com/loft-sh/vcluster-sdk/syncer/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testPreferParentExecute(
	testName string,
	testCase *testPreferParentEnvVolTestCase,
	getHook func(ctx *vclustersdksyncercontext.RegisterContext) hooks.EnvVolMutatingHook,
	getActual func(resPod *corev1.Pod) string,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s: starting", testName)

		scheme := newScheme()

		pClient := vclustersdksyncertesting.NewFakeClient(scheme, testCase.pClientObjs...)
		vClient := vclustersdksyncertesting.NewFakeClient(scheme, testCase.vClientObjs...)

		ctx := vclustersdksyncertesting.NewFakeRegisterContext(pClient, vClient)

		h := getHook(ctx)

		res, err := h.MutateCreatePhysical(context.Background(), testCase.mutateObj)
		if err != nil {
			t.Fatal(err)
		}

		resPod := res.(*corev1.Pod)

		actual := getActual(resPod)

		if actual != testCase.expected {
			t.Fatalf("got '%s', want '%s'", actual, testCase.expected)
		}
	}
}

func testPreferParentConfigmapsMutateUpdatePhysical(
	testName string,
	testCase *comparePodTestCase,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s: starting", testName)

		scheme := newScheme()

		pClient := vclustersdksyncertesting.NewFakeClient(scheme)
		vClient := vclustersdksyncertesting.NewFakeClient(scheme)

		ctx := vclustersdksyncertesting.NewFakeRegisterContext(pClient, vClient)

		h := hooks.NewPreferParentConfigmapsHook(ctx)

		res, err := h.MutateUpdatePhysical(context.Background(), testCase.inPod)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(res, testCase.expected) {
			t.Fatalf(
				"%s: actual and expected inputs do not match\nactual: %s\nexpected:%s",
				testName,
				testCase.inPod,
				testCase.expected,
			)
		}
	}
}

func TestPreferParentConfigmapsMutateUpdatePhysical(t *testing.T) {
	cases := map[string]*comparePodTestCase{
		"no-existing-annotations": {
			inPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expected: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"vcluster.loft.sh/mutated-by-hook": "prefer-parent-configmaps-hook",
					},
				},
			},
		},
		"existing-annotations": {
			inPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"someannotation": "somevalue",
					},
				},
			},
			expected: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"someannotation":                   "somevalue",
						"vcluster.loft.sh/mutated-by-hook": "prefer-parent-configmaps-hook",
					},
				},
			},
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentConfigmapsMutateUpdatePhysical(testName, testCase)
		t.Run(testName, f)
	}
}
