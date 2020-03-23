module sqs-autoscaler-controller

go 1.14

require (
	github.com/aws/aws-sdk-go v1.29.29
	github.com/pubnub/go-metrics-statsd v0.0.0-20170124014003-7da61f429d6b
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/sirupsen/logrus v1.4.2
	github.com/vmg/backoff v1.0.0
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v0.17.0
)
