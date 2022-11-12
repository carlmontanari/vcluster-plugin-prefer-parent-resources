package hooks_test

import (
	"testing"

	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/prefer-parent-resources/hooks"
	vclustersdksyncertranslator "github.com/loft-sh/vcluster-sdk/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	somesecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "somesecret",
			Namespace: "test",
		},
		Data: map[string][]byte{"somekey": []byte("someval")},
	}
	somepodWithSecretVolume = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "somepod",
			Namespace: "test",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name: "",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "somesecret",
					},
				},
			}},
		},
		Status: corev1.PodStatus{},
	}
	somepodWithSecretEnv = &corev1.Pod{
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
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "somesecret",
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
	}
)

func TestPreferParentSecretsVolumesMutateCreatePhysical(t *testing.T) {
	cases := map[string]*testPreferParentEnvVolTestCase{
		"no-sync-annotation": {
			description: "validate that pods with the 'no-sync' annotation do not get mutated " +
				"to attach to 'real' secret",
			pClientObjs: []runtime.Object{somesecret},
			vClientObjs: []runtime.Object{somepodWithSecretVolume},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
						hooks.SkipPreferSecretsHook:                     "1",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "somesecret-x-test-x-suffix",
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			volPos:   0,
			expected: "somesecret-x-test-x-suffix",
		},
		"no-sync-no-real-secret-as-volume": {
			description: "validate that pods with a 'not real' secret mounted as a volume end  " +
				"up using the 'virtual' (vcluster) secret",
			pClientObjs: []runtime.Object{},
			vClientObjs: []runtime.Object{somepodWithSecretVolume},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "somesecret-x-test-x-suffix",
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			volPos:   0,
			expected: "somesecret-x-test-x-suffix",
		},
		"sync-real-secret-as-volume": {
			description: "validate that pods with a 'real' secret mounted as a volume end up " +
				"using the 'parent' (pcluster) secret",
			pClientObjs: []runtime.Object{somesecret},
			vClientObjs: []runtime.Object{somepodWithSecretVolume},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "somesecret-x-test-x-suffix",
							},
						},
					}},
				},
				Status: corev1.PodStatus{},
			},
			volPos:   0,
			expected: "somesecret",
		},
		"sync-real-secret-as-volume-non-zero-volume-pos": {
			description: "validate that pods with a 'real' secret mounted as an volume (in not zero" +
				"position) end up using the 'parent' (pcluster) secret",
			pClientObjs: []runtime.Object{somesecret},
			vClientObjs: []runtime.Object{&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "someothersecret",
								},
							},
						},
						{
							Name: "",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "somesecret",
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{},
			}},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "someothersecret-x-test-x-suffix",
								},
							},
						},
						{
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "somesecret-x-test-x-suffix",
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{},
			},
			volPos:   1,
			expected: "somesecret",
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentExecute(
			testName,
			testCase,
			hooks.NewPreferParentSecretsHook,
			func(resPod *corev1.Pod) string {
				volPos := testCase.volPos

				return resPod.Spec.Volumes[volPos].VolumeSource.Secret.SecretName
			},
		)
		t.Run(testName, f)
	}
}

func TestPreferParentSecretsEnvVarMutateCreatePhysical(t *testing.T) {
	cases := map[string]*testPreferParentEnvVolTestCase{
		"no-sync-annotation": {
			description: "validate that pods with the 'no-sync' annotation do not get mutated " +
				"to attach to 'real' configmap",
			pClientObjs: []runtime.Object{somesecret},
			vClientObjs: []runtime.Object{somepodWithSecretEnv},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
						hooks.SkipPreferSecretsHook:                     "1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "somecontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-real-secret",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecret-x-test-x-suffix",
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
			containerPos: 0,
			envPos:       0,
			expected:     "somesecret-x-test-x-suffix",
		},
		"no-sync-no-real-secret-as-env": {
			description: "validate that pods with a 'not real' secret mounted as an envvar end  " +
				"up using the 'virtual' (vcluster) secret",
			pClientObjs: []runtime.Object{},
			vClientObjs: []runtime.Object{somepodWithSecretEnv},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "somecontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-real-secret",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecret-x-test-x-suffix",
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
			containerPos: 0,
			envPos:       0,
			expected:     "somesecret-x-test-x-suffix",
		},
		"sync-real-secret-as-env": {
			description: "validate that pods with a 'real' secret mounted as an envvar end up " +
				"using the 'parent' (pcluster) secret",
			pClientObjs: []runtime.Object{somesecret},
			vClientObjs: []runtime.Object{somepodWithSecretEnv},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
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
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecret-x-test-x-suffix",
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
			containerPos: 0,
			envPos:       0,
			expected:     "somesecret",
		},
		"sync-real-secret-as-env-non-zero-env-pos": {
			description: "validate that pods with a 'real' secret mounted as an envvar (in not zero" +
				"position) end up using the 'parent' (pcluster) secret",
			pClientObjs: []runtime.Object{somesecret},
			vClientObjs: []runtime.Object{&corev1.Pod{
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
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecretnotreal",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
								{
									Name: "env-from-real-configmap",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecret",
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
			}},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "somecontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-not-real-configmap",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecretnotreal-x-test-x-suffix",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
								{
									Name: "env-from-real-configmap",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecret-x-test-x-suffix",
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
			containerPos: 0,
			envPos:       1,
			expected:     "somesecret",
		},
		"sync-real-secret-as-env-non-zero-container-pos": {
			description: "validate that pods with a 'real' secret mounted as an envvar (in not zero" +
				"position) end up using the 'parent' (pcluster) secret",
			pClientObjs: []runtime.Object{somesecret},
			vClientObjs: []runtime.Object{&corev1.Pod{
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
									Name: "env-from-not-real-secret",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecretnotreal",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
							},
						},
						{
							Name:  "anothercontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-real-secret",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecret",
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
			}},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "somecontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-not-real-secret",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecretnotreal-x-test-x-suffix",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
							},
						},
						{
							Name:  "anothercontainer",
							Image: "someimage:latest",
							Env: []corev1.EnvVar{
								{
									Name: "env-from-real-secret",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "somesecret",
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
			containerPos: 1,
			envPos:       0,
			expected:     "somesecret",
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentExecute(
			testName,
			testCase,
			hooks.NewPreferParentSecretsHook,
			func(resPod *corev1.Pod) string {
				containerPos := testCase.containerPos
				envPos := testCase.envPos

				return resPod.Spec.Containers[containerPos].Env[envPos].
					ValueFrom.SecretKeyRef.LocalObjectReference.Name
			},
		)
		t.Run(testName, f)
	}
}
