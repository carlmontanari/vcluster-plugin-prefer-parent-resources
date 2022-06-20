package hooks_test

import (
	"testing"

	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/hooks"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testMutateAnnotations(testName string, testCase *comparePodTestCase) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s: starting", testName)

		hooks.MutateAnnotations(testCase.inPod, "prefer-parent-configmaps-hook")

		if !cmp.Equal(testCase.inPod, testCase.expected) {
			t.Fatalf(
				"%s: actual and expected inputs do not match\nactual: %s\nexpected:%s",
				testName,
				testCase.inPod,
				testCase.expected,
			)
		}
	}
}

func TestMutateAnnotations(t *testing.T) {
	cases := map[string]*comparePodTestCase{
		"no-existing-annotations": {
			inPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
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
		f := testMutateAnnotations(testName, testCase)
		t.Run(testName, f)
	}
}
