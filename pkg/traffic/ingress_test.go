package traffic_test

import (
	"testing"

	"github.com/kuadrant/kcp-glbc/pkg/traffic"
	testSupport "github.com/kuadrant/kcp-glbc/test/support"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyTransformsIngress(t *testing.T) {
	cases := []struct {
		Name string
		// OriginalIngress is the ingress as the user created it
		OriginalIngress *networkingv1.Ingress
		// ReconciledIngress is the ingress after the controller has done its work and ready to save it
		ReconciledIngress *networkingv1.Ingress
		ExpectErr         bool
	}{
		{
			Name: "test original spec not changed post reconcile and transforms applied single host",
			OriginalIngress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"experimental.status.workload.kcp.dev/c1": "",
					},
					Labels: map[string]string{
						"state.workload.kcp.dev/c1": "Sync",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "test"}}},
									},
								},
							},
						},
					},
				},
			},
			ReconciledIngress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"experimental.status.workload.kcp.dev/c1": "",
						"experimental.status.workload.kcp.dev/c2": "",
					},
					Labels: map[string]string{
						"state.workload.kcp.dev/c1": "Sync",
						"state.workload.kcp.dev/c2": "Sync",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "guid.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "test"}}},
									},
								},
							},
						},
					},
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"guid.example.com"}, SecretName: "test"},
					},
				},
			},
		},
		{
			Name: "test original spec not changed post reconcile and transforms applied multiple verified hosts",
			OriginalIngress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"experimental.status.workload.kcp.dev/c1": "",
					},
					Labels: map[string]string{
						"state.workload.kcp.dev/c1": "Sync",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "test"}}},
									},
								},
							},
						},
						{
							Host: "test2.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "test"}}},
									},
								},
							},
						},
					},
				},
			},
			ReconciledIngress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"experimental.status.workload.kcp.dev/c1": "",
						"experimental.status.workload.kcp.dev/c2": "",
					},
					Labels: map[string]string{
						"state.workload.kcp.dev/c1": "Sync",
						"state.workload.kcp.dev/c2": "Sync",
					},
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "test"}}},
									},
								},
							},
						},
						{
							Host: "api.test.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "test"}}},
									},
								},
							},
						},
						{
							Host: "guid.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "test"}}},
									},
								},
							},
						},
					},
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"guid.example.com"}, SecretName: "glbc"},
						{Hosts: []string{"test.com", "api.test.com"}, SecretName: "test"},
					},
				},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.Name, func(t *testing.T) {
			// take a copy before we apply transforms this will have all the expected changes to the spec
			transformedCopy := testCase.ReconciledIngress.DeepCopy()
			reconciled := traffic.NewIngress(testCase.ReconciledIngress)
			original := traffic.NewIngress(testCase.OriginalIngress)
			// Apply transforms, this will reset the spec to the original once done
			err := reconciled.Transform(original)
			// after the transform is done, we should have the specs of the original and transformed remain the same
			if !equality.Semantic.DeepEqual(testCase.OriginalIngress.Spec, testCase.ReconciledIngress.Spec) {
				t.Fatalf("expected the spec of the orignal and transformed to have remained the same. Expected %v Got %v", testCase.OriginalIngress.Spec, testCase.ReconciledIngress.Spec)
			}
			// we should now have annotations applying the transforms. Validate the transformed spec matches the transform annotations.
			if err := testSupport.ValidateTransformedIngress(transformedCopy.Spec, reconciled); err != nil {
				t.Fatalf("transforms were invalid %s", err)
			}
			if testCase.ExpectErr {
				if err == nil {
					t.Fatalf("expected an error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect an error but got %v", err)
				}
			}
		})
	}

}
