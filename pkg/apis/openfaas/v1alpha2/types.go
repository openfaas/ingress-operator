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

	// Path such as /v1/profiles/view/(.*), or leave empty for default
	Path string `json:"path"`

	// IngressType such as "nginx"
	IngressType string `json:"ingressType,omitempty"`

	// Enable TLS via cert-manager
	TLS *FunctionIngressTLS `json:"tls,omitempty"`
}

// FunctionIngressTLS TLS options
type FunctionIngressTLS struct {
	Enabled bool `json:"enabled"`

	IssuerRef ObjectReference `json:"issuerRef"`
}

// UseTLS if TLS is enabled
func (f *FunctionIngressSpec) UseTLS() bool {
	return f.TLS != nil && f.TLS.Enabled
}

// ObjectReference is a reference to an object with a given name and kind.
type ObjectReference struct {
	Name string `json:"name"`

	// +optional
	Kind string `json:"kind,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FunctionIngressList is a list of Function resources
type FunctionIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []FunctionIngress `json:"items"`
}
