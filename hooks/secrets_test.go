package hooks_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/loft-sh/vcluster-sdk/syncer/translator"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/hooks"
	generictesting "github.com/loft-sh/vcluster-sdk/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type testPreferParentSecretHookTestCase struct {
	description string
	pClientObjs []runtime.Object
	vClientObjs []runtime.Object
	mutateObj   client.Object
	expected    string
}

func testPreferParentSecretsMutateCreatePhysical(
	testName string,
	testCase *testPreferParentSecretHookTestCase,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s: starting", testName)

		scheme := testingutil.NewScheme()

		pClient := generictesting.NewFakeClient(scheme, testCase.pClientObjs...)
		vClient := generictesting.NewFakeClient(scheme, testCase.vClientObjs...)

		ctx := generictesting.NewFakeRegisterContext(pClient, vClient)

		h := hooks.NewPreferParentSecretsHook(ctx).(*hooks.PreferParentSecretsHook)

		res, err := h.MutateCreatePhysical(context.Background(), testCase.mutateObj)
		if err != nil {
			t.Fatal(err)
		}

		resPod := res.(*corev1.Pod)

		actual := resPod.Spec.Volumes[0].VolumeSource.Secret.Name

		if actual != testCase.expected {
			t.Fatalf("got '%s', want '%s'", actual, testCase.expected)
		}
	}
}

func TestPreferParentSecretsMutateCreatePhysical(t *testing.T) {
	cases := map[string]*testPreferParentSecretHookTestCase{
		"no-sync-annotation": {
			description: "validate that pods with the 'no-sync' annotation do not get mutated to attach to 'real' secret",
			pClientObjs: []runtime.Object{},
			vClientObjs: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "somepod",
						Namespace: "test",
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{{
							Name: "",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "somesecret",
									},
								},
							},
						}},
					},
					Status: corev1.PodStatus{},
				},
			},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						translator.NameAnnotation:      "somepod",
						translator.NamespaceAnnotation: "test",
						hooks.SkipPreferSecretsHook:    "1",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "somesecret-x-test-x-suffix",
								},
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			expected: "somesecret-x-test-x-suffix",
		},
		"no-sync-no-real-secret": {
			description: "validate that pods with no 'real' secret end up using the 'virtual' (vcluster) secret",
			pClientObjs: []runtime.Object{},
			vClientObjs: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "somepod",
						Namespace: "test",
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{{
							Name: "",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "somesecret",
									},
								},
							},
						}},
					},
					Status: corev1.PodStatus{},
				},
			},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						translator.NameAnnotation:      "somepod",
						translator.NamespaceAnnotation: "test",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "somesecret-x-test-x-suffix",
								},
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			expected: "somesecret-x-test-x-suffix",
		},
		"sync-real-secret": {
			description: "validate that pods with a 'real' secret end up using the 'parent' (pcluster) secret",
			pClientObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "somesecret",
						Namespace: "test",
					},
					Data: map[string]string{"somekey": "someval"},
				},
			},
			vClientObjs: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "somepod",
						Namespace: "test",
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{{
							Name: "",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "somesecret",
									},
								},
							},
						}},
					},
					Status: corev1.PodStatus{},
				},
			},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						translator.NameAnnotation:      "somepod",
						translator.NamespaceAnnotation: "test",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "somesecret-x-test-x-suffix",
								},
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			expected: "somesecret",
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentSecretsMutateCreatePhysical(testName, testCase)
		t.Run(testName, f)
	}
}

func testPreferParentSecretsMutateUpdatePhysical(
	testName string,
	testCase *comparePodTestCase,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s: starting", testName)

		scheme := testingutil.NewScheme()

		pClient := generictesting.NewFakeClient(scheme)
		vClient := generictesting.NewFakeClient(scheme)

		ctx := generictesting.NewFakeRegisterContext(pClient, vClient)

		h := hooks.NewPreferParentSecretsHook(ctx).(*hooks.PreferParentSecretsHook)

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

func TestPreferParentSecretsMutateUpdatePhysical(t *testing.T) {
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
						"vcluster.loft.sh/mutated-by-hook": "prefer-parent-secrets-hook",
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
						"vcluster.loft.sh/mutated-by-hook": "prefer-parent-secrets-hook",
					},
				},
			},
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentSecretsMutateUpdatePhysical(testName, testCase)
		t.Run(testName, f)
	}
}
