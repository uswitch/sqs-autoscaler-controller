package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/uswitch/sqs-autoscaler-controller/pkg/scaler"
	"github.com/uswitch/sqs-autoscaler-controller/pkg/tpr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kubecfg = "/home/tom/.kube/config"
)

func main() {
	// log.SetLevel(log.DebugLevel)
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	cache := tpr.NewCache(sc, time.Second*10)
	go cache.Run(ctx)

	s := scaler.New(c, cache.Store, time.Second*10)
	go s.Run(ctx)

	<-stopChan
	log.Infoln("Stopped.")
}
