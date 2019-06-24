package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Function describes an OpenFaaS function
type FunctionIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FunctionIngressSpec `json:"spec"`
}

// FunctionIngressSpec is the spec for a FunctionIngress resource
type FunctionIngressSpec struct {
	Name     string `json:"name"`
	Domain   string `json:"domain"`
	Function string `json:"function"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FunctionIngressList is a list of Function resources
type FunctionIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []FunctionIngress `json:"items"`
}
