package workload

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	workload "github.com/kcp-dev/kcp/pkg/apis/workload/v1alpha1"
	"github.com/kuadrant/kcp-glbc/pkg/_internal/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	WorkloadTargetLabel          = "state.workload.kcp.dev/"
	SyncerFinalizer              = "workload.kcp.dev/syncer-"
	WorkloadClusterSoftFinalizer = "finalizers.workload.kcp.dev"
	WorkloadStatusAnnotation     = "experimental.status.workload.kcp.dev/"
	WorkloadDeletingAnnotation   = workload.InternalClusterDeletionTimestampAnnotationPrefix
	SoftFinalizer                = "kuadrant.dev/glbc-migration"
	DeleteAtAnnotation           = "kuadrant.dev/glbc-delete-at"
	TTL                          = 60
)

// Process this is a temporary solution for advanced scheduling. It will add soft finalizer annotations to a set of objects to delay their deletion.
// These are only paid attention to if advanced scheduling is on for a synctarget
func Migrate(obj metav1.Object, queue workqueue.RateLimitingInterface, logger logr.Logger) {

	ensureSoftFinalizers(obj, logger)

	gracefulRemoveSoftFinalizers(obj, queue, logger)
}

// ensureSoftFinalizers ensure all active workload clusters have a soft finalizer set
func ensureSoftFinalizers(obj metav1.Object, logger logr.Logger) {
	_, labels := metadata.HasLabelsContaining(obj, WorkloadTargetLabel)
	for label := range labels {
		labelParts := strings.Split(label, "/")
		if len(labelParts) < 2 {
			logger.Error(errors.New("invalid workload target label"), "cannot process workload migration", "label", label)
			continue
		}
		clusterName := labelParts[1]
		deleting := metadata.HasAnnotation(obj, WorkloadDeletingAnnotation+clusterName)
		if !deleting {
			softFinalizer := WorkloadClusterSoftFinalizer + "/" + clusterName
			metadata.AddAnnotation(obj, softFinalizer, SoftFinalizer)
			//if delayed delete is active on this object, remove it
			metadata.RemoveAnnotation(obj, DeleteAtAnnotation+"-"+clusterName)
		}
	}
}

// gracefulRemoveSoftFinalizers any soft finalizers with no active workload cluster should trigger a delayed delete
func gracefulRemoveSoftFinalizers(obj metav1.Object, queue workqueue.RateLimitingInterface, logger logr.Logger) {
	at := time.Now()
	at = at.Add((TTL * time.Second) * 2)
	_, annotations := metadata.HasAnnotationsContaining(obj, WorkloadClusterSoftFinalizer)
	for annotation := range annotations {
		finalizerParts := strings.Split(annotation, "/")
		if len(finalizerParts) < 2 {
			logger.Error(errors.New("invalid workload cluster soft finalizer"), "cannot process workload migration")
			continue
		}
		clusterName := finalizerParts[1]
		//finalizer on a cluster waiting to delete, set up graceful delete
		if !metadata.HasAnnotation(obj, WorkloadDeletingAnnotation+clusterName) {
			continue
		}
		clusterDeleteAtAnnotation := DeleteAtAnnotation + "-" + clusterName
		//delete delay annotation not yet set, set it
		if !metadata.HasAnnotation(obj, clusterDeleteAtAnnotation) {
			metadata.AddAnnotation(obj, clusterDeleteAtAnnotation, strconv.FormatInt(at.Unix(), 10))
			// requeue object
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				return
			}
			queue.AddAfter(key, TTL*2*time.Second)
		} else {
			deleteAt, err := strconv.Atoi(obj.GetAnnotations()[clusterDeleteAtAnnotation])
			if err != nil {
				//badly formed deleteAt annotation, remove it, so it will be regenerated
				metadata.RemoveAnnotation(obj, clusterDeleteAtAnnotation)
			}

			if int64(deleteAt) <= time.Now().Unix() {
				metadata.RemoveAnnotation(obj, WorkloadClusterSoftFinalizer+"/"+clusterName)
				metadata.RemoveAnnotation(obj, clusterDeleteAtAnnotation)
			} else {
				//requeue object
				queueFor := int64(deleteAt) - time.Now().Unix()
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err != nil {
					return
				}
				queue.AddAfter(key, time.Duration(queueFor)*time.Second)
			}
		}
	}
}
