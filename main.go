package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/uswitch/sqs-autoscaler-controller/pkg/scaler"
	"github.com/uswitch/sqs-autoscaler-controller/pkg/tpr"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kubecfg = "/home/tom/.kube/config"
)

type options struct {
	kubeconfig string
}

func createClient(opts *options) (*kubernetes.Clientset, *rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubecfg)
	if err != nil {
		return nil, nil, err
	}
	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	return c, config, nil
}

func main() {
	opts := &options{}
	kingpin.Flag("kubeconfig", "Path to kubeconfig.").StringVar(&opts.kubeconfig)

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, config, err := createClient(opts)
	if err != nil {
		log.Fatalf("error creating client: %s", err)
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
