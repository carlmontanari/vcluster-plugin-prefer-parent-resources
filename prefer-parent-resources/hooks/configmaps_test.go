package hooks_test

import (
	"context"
	"testing"

	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/prefer-parent-resources/hooks"

	"github.com/google/go-cmp/cmp"

	"github.com/loft-sh/vcluster-sdk/syncer/translator"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	generictesting "github.com/loft-sh/vcluster-sdk/syncer/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func falsePtr() *bool {
	f := false

	return &f
}

type testPreferParentConfigmapHookTestCase struct {
	description string
	pClientObjs []runtime.Object
	vClientObjs []runtime.Object
	mutateObj   ctrlruntimeclient.Object
	expected    string
}

func testPreferParentConfigmapsVolumesMutateCreatePhysical(
	testName string,
	testCase *testPreferParentConfigmapHookTestCase,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s: starting", testName)

		scheme := newScheme()

		pClient := generictesting.NewFakeClient(scheme, testCase.pClientObjs...)
		vClient := generictesting.NewFakeClient(scheme, testCase.vClientObjs...)

		ctx := generictesting.NewFakeRegisterContext(pClient, vClient)

		h := hooks.NewPreferParentConfigmapsHook(ctx).(*hooks.PreferParentConfigmapsHook)

		res, err := h.MutateCreatePhysical(context.Background(), testCase.mutateObj)
		if err != nil {
			t.Fatal(err)
		}

		resPod := res.(*corev1.Pod)

		actual := resPod.Spec.Volumes[0].VolumeSource.ConfigMap.Name

		if actual != testCase.expected {
			t.Fatalf("got '%s', want '%s'", actual, testCase.expected)
		}
	}
}

func TestPreferParentConfigmapsVolumesMutateCreatePhysical(t *testing.T) {
	cases := map[string]*testPreferParentConfigmapHookTestCase{
		"no-sync-annotation": {
			description: "validate that pods with the 'no-sync' annotation do not get mutated " +
				"to attach to 'real' configmap",
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
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "someconfigmap",
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
						hooks.SkipPreferConfigMapsHook: "1",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "someconfigmap-x-test-x-suffix",
								},
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			expected: "someconfigmap-x-test-x-suffix",
		},
		"no-sync-no-real-configmap-as-volume": {
			description: "validate that pods with a 'not real' configmap mounted as a volume end  " +
				"up using the 'virtual' (vcluster) configmap",
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
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "someconfigmap",
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
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "someconfigmap-x-test-x-suffix",
								},
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			expected: "someconfigmap-x-test-x-suffix",
		},
		"sync-real-configmap-as-volume": {
			description: "validate that pods with a 'real' configmap mounted as a volume end up " +
				"using the 'parent' (pcluster) configmap",
			pClientObjs: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "someconfigmap",
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
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "someconfigmap",
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
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "someconfigmap-x-test-x-suffix",
								},
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			expected: "someconfigmap",
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentConfigmapsVolumesMutateCreatePhysical(testName, testCase)
		t.Run(testName, f)
	}
}

func testPreferParentConfigmapsEnvVarMutateCreatePhysical(
	testName string,
	testCase *testPreferParentConfigmapHookTestCase,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s: starting", testName)

		scheme := newScheme()

		pClient := generictesting.NewFakeClient(scheme, testCase.pClientObjs...)
		vClient := generictesting.NewFakeClient(scheme, testCase.vClientObjs...)

		ctx := generictesting.NewFakeRegisterContext(pClient, vClient)

		h := hooks.NewPreferParentConfigmapsHook(ctx).(*hooks.PreferParentConfigmapsHook)

		res, err := h.MutateCreatePhysical(context.Background(), testCase.mutateObj)
		if err != nil {
			t.Fatal(err)
		}

		resPod := res.(*corev1.Pod)

		actual := resPod.Spec.Containers[0].Env[0].ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name

		if actual != testCase.expected {
			t.Fatalf("got '%s', want '%s'", actual, testCase.expected)
		}
	}
}

func TestPreferParentConfigmapsEnvVarMutateCreatePhysical(t *testing.T) {
	cases := map[string]*testPreferParentConfigmapHookTestCase{
		"no-sync-annotation": {
			description: "validate that pods with the 'no-sync' annotation do not get mutated " +
				"to attach to 'real' configmap",
			pClientObjs: []runtime.Object{},
			vClientObjs: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "somepod",
						Namespace: "test",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "somecontainer",
								Image: "someimage:latest",
								Env: []corev1.EnvVar{
									{
										Name: "env-from-real-configmap",
										ValueFrom: &corev1.EnvVarSource{
											ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "someconfigmap",
												},
												Key:      "somekey",
												Optional: falsePtr(),
											},
										},
									},
								},
							},
						},
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
						hooks.SkipPreferConfigMapsHook: "1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "somecontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-real-configmap",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "someconfigmap-x-test-x-suffix",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{},
			},
			expected: "someconfigmap-x-test-x-suffix",
		},
		"no-sync-no-real-configmap-as-volume": {
			description: "validate that pods with a 'not real' configmap mounted as an envvar end " +
				"up using the 'virtual' (vcluster) configmap",
			pClientObjs: []runtime.Object{},
			vClientObjs: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "somepod",
						Namespace: "test",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "somecontainer",
								Image: "someimage:latest",
								Env: []corev1.EnvVar{
									{
										Name: "env-from-real-configmap",
										ValueFrom: &corev1.EnvVarSource{
											ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "someconfigmap",
												},
												Key:      "somekey",
												Optional: falsePtr(),
											},
										},
									},
								},
							},
						},
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
					Containers: []corev1.Container{
						{
							Name:  "somecontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-real-configmap",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "someconfigmap-x-test-x-suffix",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{},
			},
			expected: "someconfigmap-x-test-x-suffix",
		},
		"sync-real-configmap-as-env": {
			description: "validate that pods with a 'real' configmap mounted as an envvar end up " +
				"using the 'parent' (pcluster) configmap",
			pClientObjs: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "someconfigmap",
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
						Containers: []corev1.Container{
							{
								Name:  "somecontainer",
								Image: "someimage:latest",
								Env: []corev1.EnvVar{
									{
										Name: "env-from-real-configmap",
										ValueFrom: &corev1.EnvVarSource{
											ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "someconfigmap",
												},
												Key:      "somekey",
												Optional: falsePtr(),
											},
										},
									},
								},
							},
						},
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
					Containers: []corev1.Container{
						{
							Name:  "somecontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-real-configmap",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "someconfigmap-x-test-x-suffix",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{},
			},
			expected: "someconfigmap",
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentConfigmapsEnvVarMutateCreatePhysical(testName, testCase)
		t.Run(testName, f)
	}
}

func testPreferParentConfigmapsMutateUpdatePhysical(
	testName string,
	testCase *comparePodTestCase,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s: starting", testName)

		scheme := newScheme()

		pClient := generictesting.NewFakeClient(scheme)
		vClient := generictesting.NewFakeClient(scheme)

		ctx := generictesting.NewFakeRegisterContext(pClient, vClient)

		h := hooks.NewPreferParentConfigmapsHook(ctx).(*hooks.PreferParentConfigmapsHook)

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
