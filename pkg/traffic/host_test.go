package traffic

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/kuadrant/kcp-glbc/pkg/_internal/metadata"
	v1 "github.com/kuadrant/kcp-glbc/pkg/apis/kuadrant/v1"
)

type hostResult struct {
	Status   ReconcileStatus
	Err      error
	Accessor Interface
}

func TestReconcileHost(t *testing.T) {
	accessor := func(rules []networkingv1.IngressRule, tls []networkingv1.IngressTLS) Interface {
		i := &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				Rules: rules,
			},
		}
		i.Spec.TLS = tls

		return &Ingress{Ingress: i}
	}

	var buildResult = func(r Reconciler, a Interface) hostResult {
		status, err := r.Reconcile(context.TODO(), a)
		return hostResult{
			Status:   status,
			Err:      err,
			Accessor: a,
		}
	}
	var managedDomain = "test.com"

	var commonValidation = func(hr hostResult, expectedStatus ReconcileStatus) error {
		if hr.Status != expectedStatus {
			return fmt.Errorf("unexpected status ")
		}
		if hr.Err != nil {
			return fmt.Errorf("unexpected error from Reconcile : %s", hr.Err)
		}
		if !metadata.HasAnnotation(hr.Accessor, ANNOTATION_HCG_HOST) {
			return fmt.Errorf("expected annotation %s to be set", ANNOTATION_HCG_HOST)
		}
		return nil
	}

	cases := []struct {
		Name     string
		Accessor func() Interface
		Validate func(hr hostResult) error
	}{
		{
			Name: "test managed host generated for empty host field",
			Accessor: func() Interface {
				return accessor([]networkingv1.IngressRule{{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{},
					},
				}, {
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{},
					},
				}}, []networkingv1.IngressTLS{})

			},
			Validate: func(hr hostResult) error {
				return commonValidation(hr, ReconcileStatusStop)
			},
		},
		{
			Name: "test custom host replaced with generated managed host",
			Accessor: func() Interface {
				a := accessor([]networkingv1.IngressRule{{
					Host: "api.example.com",
				}}, []networkingv1.IngressTLS{})

				metadata.AddAnnotation(a, ANNOTATION_HCG_HOST, "123.test.com")

				return a
			},
			Validate: func(hr hostResult) error {
				err := commonValidation(hr, ReconcileStatusContinue)
				if err != nil {
					return err
				}
				if !metadata.HasAnnotation(hr.Accessor, ANNOTATION_HCG_CUSTOM_HOST_REPLACED) {
					return fmt.Errorf("expected the custom host annotation to be present")
				}
				generatedHost, ok := hr.Accessor.GetAnnotations()[ANNOTATION_HCG_HOST]
				if !ok {
					return ErrGeneratedHostMissing
				}
				for _, host := range hr.Accessor.GetHosts() {
					if host != generatedHost {
						return fmt.Errorf("expected the host to be set to %s, but got %s", generatedHost, host)
					}
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := &HostReconciler{
				ManagedDomain: managedDomain,
				GetDomainVerifications: func(ctx context.Context, accessor Interface) (*v1.DomainVerificationList, error) {
					return &v1.DomainVerificationList{}, nil
				},
			}

			if err := tc.Validate(buildResult(reconciler, tc.Accessor())); err != nil {
				t.Fatalf("fail: %s", err)
			}

		})
	}
}

func TestProcessCustomHostValidation(t *testing.T) {
	testCases := []struct {
		name                string
		accessor            Interface
		domainVerifications *v1.DomainVerificationList
		expectedRules       []networkingv1.IngressRule
		expectedTLS         []networkingv1.IngressTLS
	}{
		{
			name: "Empty host",
			accessor: &Ingress{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
						Annotations: map[string]string{
							ANNOTATION_HCG_HOST: "generated.host.net",
						},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/",
								},
							},
						},
					},
				},
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Custom host verified",
			accessor: &Ingress{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
						Annotations: map[string]string{
							ANNOTATION_HCG_HOST: "generated.host.net",
						},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "test.pb-custom.hcpapps.net",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/path",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{
				Items: []v1.DomainVerification{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pb-custom.hcpapps.net",
						},
						Spec: v1.DomainVerificationSpec{
							Domain: "pb-custom.hcpapps.net",
						},
						Status: v1.DomainVerificationStatus{
							Verified: true,
						},
					},
				},
			},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "test.pb-custom.hcpapps.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "subdomain of verifiied custom host",
			accessor: &Ingress{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
						Annotations: map[string]string{
							ANNOTATION_HCG_HOST: "generated.host.net",
						},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "sub.test.pb-custom.hcpapps.net",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/path",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{
				Items: []v1.DomainVerification{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pb-custom.hcpapps.net",
						},
						Spec: v1.DomainVerificationSpec{
							Domain: "pb-custom.hcpapps.net",
						},
						Status: v1.DomainVerificationStatus{
							Verified: true,
						},
					},
				},
			},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "sub.test.pb-custom.hcpapps.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Custom host unverified",
			accessor: &Ingress{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
						Annotations: map[string]string{
							ANNOTATION_HCG_HOST: "generated.host.net",
						},
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: "test.pb-custom.hcpapps.net",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/path",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{
				Items: []v1.DomainVerification{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pb-custom.hcpapps.net",
						},
						Spec: v1.DomainVerificationSpec{
							Domain: "pb-custom.hcpapps.net",
						},
						Status: v1.DomainVerificationStatus{
							Verified: false,
						},
					},
				},
			},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Unverfied host TLS section is removed",
			accessor: &Ingress{
				Ingress: &networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingress",
						Annotations: map[string]string{
							ANNOTATION_HCG_HOST: "generated.host.net",
						},
					},
					Spec: networkingv1.IngressSpec{
						TLS: []networkingv1.IngressTLS{
							{
								Hosts: []string{
									"test.pb-custom.hcpapps.net",
								},
								SecretName: "tls-secret",
							},
							{
								Hosts: []string{
									"generated.host.net",
								},
								SecretName: "tls-secret",
							},
						},
						Rules: []networkingv1.IngressRule{
							{
								Host: "test.pb-custom.hcpapps.net",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "/path",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			domainVerifications: &v1.DomainVerificationList{
				Items: []v1.DomainVerification{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pb-custom.hcpapps.net",
						},
						Spec: v1.DomainVerificationSpec{
							Domain: "pb-custom.hcpapps.net",
						},
						Status: v1.DomainVerificationStatus{
							Verified: false,
						},
					},
				},
			},
			expectedRules: []networkingv1.IngressRule{
				{
					Host: "generated.host.net",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/path",
								},
							},
						},
					},
				},
			},
			expectedTLS: []networkingv1.IngressTLS{
				{
					Hosts: []string{
						"generated.host.net",
					},
					SecretName: "tls-secret",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ingressAccessor := testCase.accessor.(*Ingress)
			if err := testCase.accessor.ProcessCustomHosts(
				context.TODO(),
				testCase.domainVerifications,
				func(ctx context.Context, i Interface) error {
					return nil
				},
				func(ctx context.Context, i Interface) error {
					return nil
				},
			); err != nil {
				t.Fatal(err)
			}

			// Assert the reconciled rules match the expected rules
			for _, expectedRule := range testCase.expectedRules {
				foundExpectedRule := false
				for _, rule := range ingressAccessor.Spec.Rules {
					if equality.Semantic.DeepEqual(expectedRule, rule) {
						foundExpectedRule = true
						break
					}
				}
				if !foundExpectedRule {
					t.Fatalf("Expected rule not found: %+v", expectedRule)
				}
			}

			for _, expectedTLS := range testCase.expectedTLS {
				foundExpectedTLS := false
				for _, tls := range ingressAccessor.Spec.TLS {
					if equality.Semantic.DeepEqual(expectedTLS, tls) {
						foundExpectedTLS = true
						break
					}
				}

				if !foundExpectedTLS {
					t.Fatalf("Expected TLS not found: %+v", expectedTLS)
				}
			}
		})
	}
}
