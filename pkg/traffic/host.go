package traffic

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/rs/xid"

	"github.com/kuadrant/kcp-glbc/pkg/_internal/metadata"
	v1 "github.com/kuadrant/kcp-glbc/pkg/apis/kuadrant/v1"
	kuadrantclientv1 "github.com/kuadrant/kcp-glbc/pkg/client/kuadrant/clientset/versioned"
)

type HostReconciler struct {
	ManagedDomain          string
	Log                    logr.Logger
	KuadrantClient         kuadrantclientv1.ClusterInterface
	GetDomainVerifications func(ctx context.Context, accessor Interface) (*v1.DomainVerificationList, error)
	CreateOrUpdateTraffic  CreateOrUpdateTraffic
	DeleteTraffic          DeleteTraffic
}

func (r *HostReconciler) GetName() string {
	return "Host Reconciler"
}

func (r *HostReconciler) Reconcile(ctx context.Context, accessor Interface) (ReconcileStatus, error) {
	if !metadata.HasAnnotation(accessor, ANNOTATION_HCG_HOST) {
		// Let's assign it a global hostname if any
		generatedHost := fmt.Sprintf("%s.%s", xid.New(), r.ManagedDomain)
		metadata.AddAnnotation(accessor, ANNOTATION_HCG_HOST, generatedHost)
		//we need this host set and saved on the accessor before we go any further so force an update
		// if this is not saved we end up with a new host and the certificate can have the wrong host
		return ReconcileStatusStop, nil
	}
	//TODO this should only be set once everything is ready (DNS, and Certificate)
	//https://github.com/kcp-dev/kcp-glbc/issues/399
	dvs, err := r.GetDomainVerifications(ctx, accessor)
	if err != nil {
		return ReconcileStatusContinue, fmt.Errorf("error getting domain verifications: %v", err)
	}
	err = accessor.ProcessCustomHosts(ctx, dvs, r.CreateOrUpdateTraffic, r.DeleteTraffic)
	if err != nil {
		return ReconcileStatusStop, fmt.Errorf("error processing custom hosts: %v", err)
	}
	return ReconcileStatusContinue, nil
}
