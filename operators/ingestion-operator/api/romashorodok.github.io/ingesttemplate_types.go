package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngestTemplateSpec struct {
	Image string                 `json:"image,omitempty"`
	Ports []corev1.ContainerPort `json:"ports,omitempty"`
}

type IngestTemplateStatus struct {
	Replicas int32 `json:"replicas"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type IngestTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngestTemplateSpec   `json:"spec,omitempty"`
	Status IngestTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type IngestTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngestTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IngestTemplate{}, &IngestTemplateList{})
}
