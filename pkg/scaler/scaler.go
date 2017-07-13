package scaler

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rcrowley/go-metrics"
	"github.com/uswitch/sqs-autoscaler-controller/pkg/crd"
	"github.com/vmg/backoff"
	appsv1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/aws/aws-sdk-go/aws/session"
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
			t := metrics.GetOrRegisterTimer("scaler.do", metrics.DefaultRegistry)
			t.Time(func() { s.do(ctx, sess) })
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

func scaleFields(scaler *crd.SqsAutoScaler) log.Fields {
	return log.Fields{"name": scaler.ObjectMeta.Name, "namespace": scaler.ObjectMeta.Namespace, "queue": scaler.Spec.Queue, "deploy": scaler.Spec.Deployment}
}

func (s Scaler) targetReplicas(size int64, scale *crd.SqsAutoScaler, d *appsv1.Deployment) (int32, error) {
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

func (s Scaler) executeScale(ctx context.Context, sess *session.Session, scale *crd.SqsAutoScaler) (*appsv1.Deployment, int32, error) {
	size, err := CurrentQueueSize(sess, scale.Spec.Queue)
	if err != nil {
		return nil, 0, err
	}

	deployments := s.client.Apps().Deployments(scale.ObjectMeta.Namespace)
	deployment, err := deployments.Get(scale.Spec.Deployment, metav1.GetOptions{})
	if err != nil {
		return nil, 0, err
	}

	if deployment.Status.Replicas != deployment.Status.AvailableReplicas {
		return nil, 0, fmt.Errorf("deployment available replicas not at target. won't adjust")
	}

	replicas, err := s.targetReplicas(size, scale, deployment)
	if err != nil {
		return nil, 0, err
	}

	delta := replicas - *deployment.Spec.Replicas
	deployment.Spec.Replicas = &replicas
	updated, err := deployments.Update(deployment)

	if delta != 0 && err == nil {
		metrics.GetOrRegisterMeter("scaler.adjust", metrics.DefaultRegistry).Mark(1)
	}

	return updated, delta, err
}

const (
	ReasonScaleDeployment       = "ScaleSuccess"
	ReasonFailedScaleDeployment = "ScaleFail"
)

func (s Scaler) do(ctx context.Context, sess *session.Session) {
	for _, obj := range s.store.List() {
		scaler := obj.(*crd.SqsAutoScaler)
		logger := log.WithFields(scaleFields(scaler))
		op := func() error {
			deployment, delta, err := s.executeScale(ctx, sess, scaler)
			if err != nil {
				logger.Warnf("unable to perform scale, will retry: %s", err)
				return err
			}

			logger.WithFields(log.Fields{"delta": delta, "desired": *deployment.Spec.Replicas, "available": deployment.Status.AvailableReplicas}).Info("Updated deployment")
			if delta != 0 {
				scaler.RecordEvent(s.client, crd.TypeNormal, ReasonScaleDeployment, fmt.Sprintf("adjusted deployment to %d (delta: %d)", *deployment.Spec.Replicas, delta))
			}
			return nil
		}
		strategy := backoff.NewExponentialBackOff()
		strategy.MaxInterval = time.Second
		strategy.MaxElapsedTime = time.Second * 5
		strategy.InitialInterval = time.Millisecond * 100

		err := backoff.Retry(op, strategy)
		if err != nil {
			metrics.GetOrRegisterMeter("scaler.doError", metrics.DefaultRegistry).Mark(1)

			msg := fmt.Sprintf("error scaling: %s", err)
			logger.Error(msg)

			err = scaler.RecordEvent(s.client, crd.TypeWarning, ReasonFailedScaleDeployment, msg)
			if err != nil {
				logger.Error("error recording event", err)
			}
		}
	}
}
