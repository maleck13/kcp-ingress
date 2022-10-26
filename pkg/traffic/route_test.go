package traffic_test

import (
	"testing"

	"github.com/kuadrant/kcp-glbc/pkg/traffic"
	testSupport "github.com/kuadrant/kcp-glbc/test/support"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyTransformsRoute(t *testing.T) {
	cases := []struct {
		Name string
		// OriginalIngress is the ingress as the user created it
		OriginalRoute *routev1.Route
		// ReconciledIngress is the ingress after the controller has done its work and ready to save it
		ReconciledRoute *routev1.Route
		ExpectErr       bool
	}{{
		Name: "test origin spec not modified",
		OriginalRoute: &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Annotations: map[string]string{
					"experimental.status.workload.kcp.dev/c1": "",
				},
				Labels: map[string]string{
					"state.workload.kcp.dev/c1": "Sync",
				},
			},
			Spec: routev1.RouteSpec{
				Host: "test.com",
				Path: "/",
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: "test",
				},
				TLS: &routev1.TLSConfig{
					Termination:   routev1.TLSTerminationEdge,
					Certificate:   "xyz",
					Key:           "xyz",
					CACertificate: "xyz",
				},
			},
		},
		ReconciledRoute: &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Annotations: map[string]string{
					"experimental.status.workload.kcp.dev/c1": "",
				},
				Labels: map[string]string{
					"state.workload.kcp.dev/c1": "Sync",
				},
			},
			Spec: routev1.RouteSpec{
				Host: "glbc.com",
				Path: "/",
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: "test",
				},
				TLS: &routev1.TLSConfig{
					Termination:   routev1.TLSTerminationEdge,
					Certificate:   "abc",
					Key:           "abc",
					CACertificate: "abc",
				},
			},
		},
	}}

	for _, testCase := range cases {
		t.Run(testCase.Name, func(t *testing.T) {
			// take a copy before we apply transforms this will have all the expected changes to the spec
			transformedCopy := testCase.ReconciledRoute.DeepCopy()
			reconciled := traffic.NewRoute(testCase.ReconciledRoute)
			original := traffic.NewRoute(testCase.OriginalRoute)
			// Apply transforms, this will reset the spec to the original once done
			err := reconciled.Transform(original)
			// after the transform is done, we should have the specs of the original and transformed remain the same
			if !equality.Semantic.DeepEqual(testCase.OriginalRoute.Spec, testCase.ReconciledRoute.Spec) {
				t.Fatalf("expected the spec of the orignal and transformed to have remained the same. Expected %v Got %v", testCase.OriginalRoute.Spec, testCase.ReconciledRoute.Spec)
			}
			// we should now have annotations applying the transforms. Validate the transformed spec matches the transform annotations.
			if err := testSupport.ValidateTransformedRoute(transformedCopy.Spec, reconciled); err != nil {
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
