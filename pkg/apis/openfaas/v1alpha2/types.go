package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FunctionIngress describes an OpenFaaS function
type FunctionIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FunctionIngressSpec `json:"spec"`
}

// FunctionIngressSpec is the spec for a FunctionIngress resource. It must
// be created in the same namespace as the gateway, i.e. openfaas.
type FunctionIngressSpec struct {
	// Domain such as www.openfaas.com
	Domain string `json:"domain"`

	// Function such as "nodeinfo"
	Function string `json:"function"`

	// Enable TLS via cert-manager
	TLS bool `json:"tls"`

	// IssuerRef name of ClusterIssuer or Issuer in same namespace as object
	IssuerRef string `json:"issuerRef"`

	// IssuerType such as ClusterIssuer or Issuer
	IssuerType string `json:"issuerType"`

	// IngressType such as "nginx"
	IngressType string `json:"ingressType"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FunctionIngressList is a list of Function resources
type FunctionIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []FunctionIngress `json:"items"`
}
