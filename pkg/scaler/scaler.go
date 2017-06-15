package scaler

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/uswitch/sqs-autoscaler-controller/pkg/tpr"
	"github.com/vmg/backoff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
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

func (s Scaler) executeScale(ctx context.Context, sess *session.Session, scale *tpr.SqsAutoScaler) (*appsv1.Deployment, error) {
	size, err := CurrentQueueSize(sess, scale.Spec.Queue)
	if err != nil {
		return nil, err
	}

	replicas, err := s.targetReplicas(size, scale)
	if err != nil {
		return nil, err
	}

	deployments := s.client.Apps().Deployments(scale.ObjectMeta.Namespace)
	d, err := deployments.Get(scale.Spec.Deployment, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	d.Spec.Replicas = &replicas
	return deployments.Update(d)
}

func (s Scaler) do(ctx context.Context, sess *session.Session) {
	for _, obj := range s.store.List() {
		scaler := obj.(*tpr.SqsAutoScaler)
		logger := log.WithFields(scaleFields(scaler))
		op := func() error {
			deployment, err := s.executeScale(ctx, sess, scaler)
			if err != nil {
				logger.Warnf("unable to perform scale check, will retry: %s", err)
				return err
			}

			logger.WithFields(log.Fields{"desired": *deployment.Spec.Replicas, "available": deployment.Status.AvailableReplicas}).Info("Updated deployment")
			return nil
		}
		strategy := backoff.NewExponentialBackOff()
		strategy.MaxInterval = time.Millisecond * 100
		strategy.MaxElapsedTime = time.Second * 2
		strategy.InitialInterval = time.Millisecond * 10

		err := backoff.Retry(op, strategy)
		if err != nil {
			logger.Error("unable to perform scale")
		}
	}
}
