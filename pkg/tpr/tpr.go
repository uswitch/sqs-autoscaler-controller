package tpr

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
)

func EnsureResource(client *kubernetes.Clientset) error {
	tpr := &v1beta1.ThirdPartyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sqs-auto-scaler.aws.uswitch.com",
		},
		Versions: []v1beta1.APIVersion{
			{Name: Version},
		},
		Description: "Deployment Autoscaler based on SQS queue length",
	}
	_, err := client.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
	if err != nil && apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func NewClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	builder := runtime.NewSchemeBuilder(addKnownTypes)
	err := builder.AddToScheme(scheme)
	if err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return client, scheme, err
}

func addToScheme() {
	runtime.NewSchemeBuilder(addKnownTypes)
}

const (
	GroupName = "aws.uswitch.com"
	Version   = "v1"
)

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &SqsAutoScaler{}, &SqsAutoScalerList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
