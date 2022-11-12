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
	someconfigmap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "someconfigmap",
			Namespace: "test",
		},
		Data: map[string]string{"somekey": "someval"},
	}
	somepodWithConfigmapVolume = &corev1.Pod{
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
	}
	somepodWithConfigmapEnv = &corev1.Pod{
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
	}
)

func TestPreferParentConfigmapsVolumesMutateCreatePhysical(t *testing.T) {
	cases := map[string]*testPreferParentEnvVolTestCase{
		"no-sync-annotation": {
			description: "validate that pods with the 'no-sync' annotation do not get mutated " +
				"to attach to 'real' configmap",
			pClientObjs: []runtime.Object{someconfigmap},
			vClientObjs: []runtime.Object{somepodWithConfigmapVolume},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
						hooks.SkipPreferConfigMapsHook:                  "1",
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
			volPos:   0,
			expected: "someconfigmap-x-test-x-suffix",
		},
		"no-sync-no-real-configmap-as-volume": {
			description: "validate that pods with a 'not real' configmap mounted as a volume end  " +
				"up using the 'virtual' (vcluster) configmap",
			pClientObjs: []runtime.Object{},
			vClientObjs: []runtime.Object{somepodWithConfigmapVolume},
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
			volPos:   0,
			expected: "someconfigmap-x-test-x-suffix",
		},
		"sync-real-configmap-as-volume": {
			description: "validate that pods with a 'real' configmap mounted as a volume end up " +
				"using the 'parent' (pcluster) configmap",
			pClientObjs: []runtime.Object{someconfigmap},
			vClientObjs: []runtime.Object{somepodWithConfigmapVolume},
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
			volPos:   0,
			expected: "someconfigmap",
		},
		"sync-real-configmap-as-volume-non-zero-volume-pos": {
			description: "validate that pods with a 'real' configmap mounted as an volume (in not " +
				"zero position) end up using the 'parent' (pcluster) secret",
			pClientObjs: []runtime.Object{someconfigmap},
			vClientObjs: []runtime.Object{&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "someotherconfigmap",
									},
								},
							},
						},
						{
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "someconfigmap",
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
					Volumes: []corev1.Volume{
						{
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "someotherconfigmap-x-test-x-suffix",
									},
								},
							},
						},
						{
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "someconfigmap-x-test-x-suffix",
									},
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{},
			},
			volPos:   1,
			expected: "someconfigmap",
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentExecute(
			testName,
			testCase,
			hooks.NewPreferParentConfigmapsHook,
			func(resPod *corev1.Pod) string {
				volPos := testCase.volPos

				return resPod.Spec.Volumes[volPos].VolumeSource.ConfigMap.Name
			},
		)
		t.Run(testName, f)
	}
}

func TestPreferParentConfigmapsEnvVarMutateCreatePhysical(t *testing.T) {
	cases := map[string]*testPreferParentEnvVolTestCase{
		"no-sync-annotation": {
			description: "validate that pods with the 'no-sync' annotation do not get mutated " +
				"to attach to 'real' configmap",
			pClientObjs: []runtime.Object{someconfigmap},
			vClientObjs: []runtime.Object{somepodWithConfigmapEnv},
			mutateObj: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "somepod",
					Namespace: "test",
					Annotations: map[string]string{
						vclustersdksyncertranslator.NameAnnotation:      "somepod",
						vclustersdksyncertranslator.NamespaceAnnotation: "test",
						hooks.SkipPreferConfigMapsHook:                  "1",
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
			containerPos: 0,
			envPos:       0,
			expected:     "someconfigmap-x-test-x-suffix",
		},
		"no-sync-no-real-configmap-as-volume": {
			description: "validate that pods with a 'not real' configmap mounted as an envvar end " +
				"up using the 'virtual' (vcluster) configmap",
			pClientObjs: []runtime.Object{},
			vClientObjs: []runtime.Object{somepodWithConfigmapEnv},
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
			containerPos: 0,
			envPos:       0,
			expected:     "someconfigmap-x-test-x-suffix",
		},
		"sync-real-configmap-as-env": {
			description: "validate that pods with a 'real' configmap mounted as an envvar end up " +
				"using the 'parent' (pcluster) configmap",
			pClientObjs: []runtime.Object{someconfigmap},
			vClientObjs: []runtime.Object{somepodWithConfigmapEnv},
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
			containerPos: 0,
			envPos:       0,
			expected:     "someconfigmap",
		},
		"sync-real-configmap-as-env-non-zero-env-pos": {
			description: "validate that pods with a 'real' configmap mounted as an envvar (in not zero" +
				"position) end up using the 'parent' (pcluster) configmap",
			pClientObjs: []runtime.Object{someconfigmap},
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
									Name: "env-from-not-real-configmap",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "someconfigmapnotreal",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
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
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "someconfigmapnotreal-x-test-x-suffix",
											},
											Key:      "somekey",
											Optional: falsePtr(),
										},
									},
								},
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
			containerPos: 0,
			envPos:       1,
			expected:     "someconfigmap",
		},
		"sync-real-configmap-as-env-non-zero-container-pos": {
			description: "validate that pods with a 'real' configmap mounted as an envvar (in not " +
				"zero position) end up using the 'parent' (pcluster) configmap",
			pClientObjs: []runtime.Object{someconfigmap},
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
									Name: "env-from-not-real-configmap",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "someconfigmapotreal",
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
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "someconfigmapnotreal-x-test-x-suffix",
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
			containerPos: 1,
			envPos:       0,
			expected:     "someconfigmap",
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentExecute(
			testName,
			testCase,
			hooks.NewPreferParentConfigmapsHook,
			func(resPod *corev1.Pod) string {
				containerPos := testCase.containerPos
				envPos := testCase.envPos

				return resPod.Spec.Containers[containerPos].Env[envPos].
					ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name
			},
		)
		t.Run(testName, f)
	}
}
