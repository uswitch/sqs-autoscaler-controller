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
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/uswitch/sqs-autoscaler-controller/pkg/crd"
	"github.com/uswitch/sqs-autoscaler-controller/pkg/scaler"
)

type options struct {
	jsonLog        bool
	kubeconfig     string
	syncInterval   time.Duration
	scaleInterval  time.Duration
	scaleDownDelay time.Duration
	statsD         string
	statsDInterval time.Duration
}

func createClientConfig(opts *options) (*rest.Config, error) {
	if opts.kubeconfig == "" {
		return rest.InClusterConfig()
	}
	return clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
}

func createClient(config *rest.Config) (*kubernetes.Clientset, error) {
	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func createApiExtensionsClient(config *rest.Config) (apiextensionsclient.Interface, error) {
	c, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func main() {
	opts := &options{}
	kingpin.Flag("json-log", "Emit logs as JSON").BoolVar(&opts.jsonLog)
	kingpin.Flag("kubeconfig", "Path to kubeconfig.").StringVar(&opts.kubeconfig)
	kingpin.Flag("sync-interval", "Interval to periodically refresh Scaler objects from Kubernetes.").Default("1m").DurationVar(&opts.syncInterval)
	kingpin.Flag("scale-interval", "Interval to check queue sizes and scale deployments.").Default("1m").DurationVar(&opts.scaleInterval)
	kingpin.Flag("scaledown-delay", "Delay in scaling down the pods once the scale down threshold is met.").Default("1s").DurationVar(&opts.scaleDownDelay)
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

	config, err := createClientConfig(opts)
	if err != nil {
		log.Fatalf("error creating client config: %s", err)
	}

	cs, err := createClient(config)
	if err != nil {
		log.Fatalf("error creating client: %s", err)
	}

	aec, err := createApiExtensionsClient(config)
	if err != nil {
		log.Fatalf("error creating apiExtensionsClient: %s", err)
	}

	err = crd.EnsureResource(aec)
	if err != nil {
		log.Fatalf("Error adding resource: %s", err)
	}

	sc, _, err := crd.NewClient(config)
	if err != nil {
		log.Fatalf("Error creating TPR client: %s", err)
	}

	cache := crd.NewCache(sc, opts.syncInterval)
	go cache.Run(ctx)

	s := scaler.New(cs, cache.Store, opts.scaleInterval, opts.scaleDownDelay)
	go s.Run(ctx)

	<-stopChan
	log.Infoln("Stopped.")
}
