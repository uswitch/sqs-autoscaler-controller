module github.com/uwswitch/sqs-autoscaler-controller

go 1.12

require (
	github.com/Sirupsen/logrus v1.0.6
	github.com/alecthomas/template v0.0.0-20160405071501-a0175ee3bccc // indirect
	github.com/alecthomas/units v0.0.0-20151022065526-2efee857e7cf // indirect
	github.com/aws/aws-sdk-go v1.20.19
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/pubnub/go-metrics-statsd v0.0.0-20170124014003-7da61f429d6b
	github.com/rcrowley/go-metrics v0.0.0-20190706150252-9beb055b7962
	github.com/uswitch/sqs-autoscaler-controller v0.0.0-20171027144604-6523e92beff0
	github.com/vmg/backoff v1.0.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/sys v0.0.0-20190712062909-fae7ac547cb7 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.0.0-20190712022805-31fe033ae6f9
	k8s.io/apiextensions-apiserver v0.0.0-20190712104117-cebf05d40107
	k8s.io/apimachinery v0.0.0-20190712095106-75ce4d1e60f1
	k8s.io/client-go v11.0.0+incompatible
)
