package support

import (
	"encoding/json"
	"fmt"

	workload "github.com/kcp-dev/kcp/pkg/apis/workload/v1alpha1"
	traffic "github.com/kuadrant/kcp-glbc/pkg/traffic"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func ValidateTransformedRoute(expectedSpec routev1.RouteSpec, transformed *traffic.Route) error {
	st := transformed.GetSyncTargets()
	for _, target := range st {
		// ensure each target has a transform value set and it is correct
		if _, ok := transformed.Annotations[workload.ClusterSpecDiffAnnotationPrefix+target]; !ok {
			return fmt.Errorf("expected a transformation for sync target " + target)
		}
		transforms := transformed.Annotations[workload.ClusterSpecDiffAnnotationPrefix+target]
		patches := []struct {
			Path  string      `json:"path"`
			Op    string      `json:"op"`
			Value interface{} `json:"value"`
		}{}
		if err := json.Unmarshal([]byte(transforms), &patches); err != nil {
			return fmt.Errorf("failed to unmarshal patch %s", err)
		}
		//ensure there is a rules and tls patch and they have the correct value
		hostPatch := false
		tlsPatch := false
		for _, p := range patches {
			if p.Path == "/host" {
				hostPatch = true
				host := ""
				b, err := json.Marshal(p.Value)
				if err != nil {
					return fmt.Errorf("failed to marshal host %s", err)
				}
				if err := json.Unmarshal(b, &host); err != nil {
					return err
				}
				if host != expectedSpec.Host {
					return fmt.Errorf("expected the host in the transform to match the rules in transformed route")
				}
			}
			if p.Path == "/tls" {
				tlsPatch = true
				tls := &routev1.TLSConfig{}
				b, err := json.Marshal(p.Value)
				if err != nil {
					return fmt.Errorf("failed to marshal tls %s", err)
				}
				if err := json.Unmarshal(b, tls); err != nil {
					return err
				}
				if !equality.Semantic.DeepEqual(tls, expectedSpec.TLS) {
					fmt.Printf("expected %v got %v ", tls, expectedSpec.TLS)
					return fmt.Errorf("expected the tls section in the transform to match the tls in transformed route")
				}
			}
		}
		if !hostPatch {
			return fmt.Errorf("expected to find a rules patch but one was missing")
		}
		if !tlsPatch {
			return fmt.Errorf("expected to find a tls patch but one was missing")
		}

	}
	return nil
}
