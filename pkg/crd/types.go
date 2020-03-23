package crd

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

//SqsAutoScaler ....
type SqsAutoScaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AutoScalerSpec `json:"spec"`
}

//SqsAutoScalerList ....
type SqsAutoScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []SqsAutoScaler `json:"items"`
}

//AutoScalerSpec ....
type AutoScalerSpec struct {
	Queue      string    `json:"queue"`
	Deployment string    `json:"deployment"`
	MinPods    int32     `json:"minPods"`
	MaxPods    int32     `json:"maxPods"`
	ScaleUp    ScaleSpec `json:"scaleUp"`
	ScaleDown  ScaleSpec `json:"scaleDown"`
}

//ScaleSpec ....
type ScaleSpec struct {
	Threshold int64 `json:"threshold"`
	Amount    int32 `json:"amount"`
}
