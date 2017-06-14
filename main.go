package main

import (
	"context"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/uswitch/k8s-sqs-scaler/pkg/tpr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kubecfg = "/home/tom/.kube/config"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", kubecfg)
	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating client: %s", err)
	}

	err = tpr.EnsureResource(c)
	if err != nil {
		log.Fatalf("Error adding resource: %s", err)
	}

	sc, _, err := tpr.NewClient(config)
	if err != nil {
		log.Fatalf("Error creating TPR client: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := tpr.NewCache(sc, time.Second*10)
	cache.Run(ctx)
}
