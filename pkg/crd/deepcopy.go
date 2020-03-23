package crd

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *SqsAutoScaler) DeepCopyInto(out *SqsAutoScaler) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = AutoScalerSpec{
		Queue:      in.Spec.Queue,
		Deployment: in.Spec.Deployment,
		MinPods:    in.Spec.MinPods,
		MaxPods:    in.Spec.MaxPods,
		ScaleUp:    in.Spec.ScaleUp,
		ScaleDown:  in.Spec.ScaleDown,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *SqsAutoScaler) DeepCopyObject() runtime.Object {
	out := SqsAutoScaler{}
	in.DeepCopyInto(&out)
	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *SqsAutoScalerList) DeepCopyObject() runtime.Object {
	out := SqsAutoScalerList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]SqsAutoScaler, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
