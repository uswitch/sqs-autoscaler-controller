package crd

import (
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	TypeNormal  = "Normal"
	TypeWarning = "Warning"
)

func (s *SqsAutoScaler) RecordEvent(k *kubernetes.Clientset, eventType, reason, message string) error {
	now := metav1.NewTime(time.Now())
	event := &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: s.ObjectMeta.Name,
			Namespace:    s.ObjectMeta.Namespace,
		},
		InvolvedObject: v1.ObjectReference{
			Kind:            "SqsAutoScaler",
			APIVersion:      "aws.uswitch.com/v1",
			Namespace:       s.ObjectMeta.Namespace,
			Name:            s.ObjectMeta.Name,
			UID:             s.ObjectMeta.UID,
			ResourceVersion: s.ObjectMeta.ResourceVersion,
		},
		Reason: reason,
		Source: v1.EventSource{
			Component: "sqs-autoscaler-controller",
		},
		FirstTimestamp: now,
		LastTimestamp:  now,
		Count:          1,
		Message:        message,
		Type:           eventType,
	}
	_, err := k.CoreV1().Events(s.ObjectMeta.Namespace).Create(event)

	return err
}
