package scaler

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/uswitch/k8s-sqs-scaler/pkg/tpr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Scaler struct {
	client   *kubernetes.Clientset
	store    cache.Store
	interval time.Duration
}

func New(client *kubernetes.Clientset, store cache.Store, interval time.Duration) *Scaler {
	return &Scaler{client, store, interval}
}

func (s Scaler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("Creating AWS Session: %s", err)
	}

	for {
		select {
		case <-ticker.C:
			log.Infof("Tick")
			s.do(ctx, sess)
		case <-ctx.Done():
			return nil
		}
	}
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func scaleFields(scaler *tpr.SqsAutoScaler) log.Fields {
	return log.Fields{"name": scaler.ObjectMeta.Name, "namespace": scaler.ObjectMeta.Namespace, "queue": scaler.Spec.Queue, "deploy": scaler.Spec.Deployment}
}

func (s Scaler) targetReplicas(size int64, scale *tpr.SqsAutoScaler) (int32, error) {
	d, err := s.client.Apps().Deployments(scale.ObjectMeta.Namespace).Get(scale.Spec.Deployment, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	replicas := d.Status.Replicas

	if size >= scale.Spec.ScaleUp.Threshold {
		desired := replicas + scale.Spec.ScaleUp.Amount
		return min(desired, scale.Spec.MaxPods), nil
	} else if size <= scale.Spec.ScaleDown.Threshold {
		desired := replicas - scale.Spec.ScaleDown.Amount
		return max(desired, scale.Spec.MinPods), nil
	}
	return replicas, nil
}

func (s Scaler) executeScale(ctx context.Context, sess *session.Session, scale *tpr.SqsAutoScaler) {
	logger := log.WithFields(scaleFields(scale))
	size, err := CurrentQueueSize(sess, scale.Spec.Queue)
	if err != nil {
		logger.Errorf("Error checking queue size: %s", err)
		return
	}
	logger = logger.WithField("size", size)
	logger.Infof("checking for scale action")

	replicas, err := s.targetReplicas(size, scale)
	if err != nil {
		logger.Errorf("Error checking target replicas: %s", err)
	}

	logger.Infof("Scaling to %d replicas", replicas)
	deployments := s.client.Apps().Deployments(scale.ObjectMeta.Namespace)
	d, err := deployments.Get(scale.Spec.Deployment, metav1.GetOptions{})
	if err != nil {
		logger.Errorf("error updating deployment: %s", err)
		return
	}
	d.Spec.Replicas = &replicas
	_, err = deployments.Update(d)
	if err != nil {
		logger.Errorf("error updating deployment: %s", err)
	}
}

func (s Scaler) do(ctx context.Context, sess *session.Session) {
	for _, obj := range s.store.List() {
		scaler := obj.(*tpr.SqsAutoScaler)
		s.executeScale(ctx, sess, scaler)
	}
}
