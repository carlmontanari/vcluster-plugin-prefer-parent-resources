package hooks_test

import (
	"testing"

	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/prefer-parent-resources/hooks"
	vclustersdksyncertranslator "github.com/loft-sh/vcluster-sdk/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestPreferParentSecretsVolumesMutateCreatePhysical(t *testing.T) {
	cases := map[string]*testPreferParentEnvVolTestCase{
		"no-sync-annotation": {
			description: "validate that pods with the 'no-sync' annotation do not get mutated " +
				"to attach to 'real' secret",
			pClientObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "somesecret",
						Namespace: "test",
					},
					Data: map[string][]byte{"somekey": []byte("someval")},
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
									SecretName: "somesecret",
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
			expected: "somesecret-x-test-x-suffix",
		},
		"no-sync-no-real-secret-as-volume": {
			description: "validate that pods with a 'not real' secret mounted as a volume end  " +
				"up using the 'virtual' (vcluster) secret",
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
									SecretName: "somesecret",
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
			expected: "somesecret-x-test-x-suffix",
		},
		"sync-real-secret-as-volume": {
			description: "validate that pods with a 'real' secret mounted as a volume end up " +
				"using the 'parent' (pcluster) secret",
			pClientObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "somesecret",
						Namespace: "test",
					},
					Data: map[string][]byte{"somekey": []byte("someval")},
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
									SecretName: "somesecret",
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
			expected: "somesecret",
		},
	}

	for testName, testCase := range cases {
		f := testPreferParentExecute(
			testName,
			testCase,
			func(resPod *corev1.Pod) string { return resPod.Spec.Volumes[0].VolumeSource.Secret.SecretName },
		)
		t.Run(testName, f)
	}
}

func TestPreferParentSecretsEnvVarMutateCreatePhysical(t *testing.T) {
	// TODO!
	cases := map[string]*testPreferParentEnvVolTestCase{}

	for testName, testCase := range cases {
		f := testPreferParentExecute(
			testName,
			testCase,
			func(resPod *corev1.Pod) string {
				return resPod.Spec.Containers[0].Env[0].ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name
			},
		)
		t.Run(testName, f)
	}
}
