package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pubnub/go-metrics-statsd"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/uswitch/sqs-autoscaler-controller/pkg/scaler"
	"github.com/uswitch/sqs-autoscaler-controller/pkg/tpr"
)

type options struct {
	jsonLog        bool
	kubeconfig     string
	syncInterval   time.Duration
	scaleInterval  time.Duration
	statsD         string
	statsDInterval time.Duration
}

func createClientConfig(opts *options) (*rest.Config, error) {
	if opts.kubeconfig == "" {
		return rest.InClusterConfig()
	}
	return clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
}

func createClient(opts *options) (*kubernetes.Clientset, *rest.Config, error) {
	config, err := createClientConfig(opts)
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
	kingpin.Flag("json-log", "Emit logs as JSON").BoolVar(&opts.jsonLog)
	kingpin.Flag("kubeconfig", "Path to kubeconfig.").StringVar(&opts.kubeconfig)
	kingpin.Flag("sync-interval", "Interval to periodically refresh Scaler objects from Kubernetes.").Default("1m").DurationVar(&opts.syncInterval)
	kingpin.Flag("scale-interval", "Interval to check queue sizes and scale deployments.").Default("1m").DurationVar(&opts.scaleInterval)

	kingpin.Flag("statsd", "UDP address to publish StatsD metrics. e.g. 127.0.0.1:8125").Default("").StringVar(&opts.statsD)
	kingpin.Flag("statsd-interval", "Interval to publish to StatsD").Default("10s").DurationVar(&opts.statsDInterval)

	kingpin.Parse()

	if opts.jsonLog {
		log.SetFormatter(&log.JSONFormatter{})
	}

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if opts.statsD != "" {
		addr, err := net.ResolveUDPAddr("udp", opts.statsD)
		if err != nil {
			log.Fatal("error parsing statsd address:", err.Error())
		}
		go statsd.StatsD(metrics.DefaultRegistry, opts.statsDInterval, "sqs-autoscaler-controller", addr)
	}

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

	cache := tpr.NewCache(sc, opts.syncInterval)
	go cache.Run(ctx)

	s := scaler.New(c, cache.Store, opts.scaleInterval)
	go s.Run(ctx)

	<-stopChan
	log.Infoln("Stopped.")
}
