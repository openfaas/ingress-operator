/*
Copyright 2023 OpenFaaS Author(s)

Licensed under the MIT license. See LICENSE file in the project root for full license information.
*/

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package applyconfiguration

import (
	v1 "github.com/openfaas/ingress-operator/pkg/apis/openfaas/v1"
	openfaasv1 "github.com/openfaas/ingress-operator/pkg/client/applyconfiguration/openfaas/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=openfaas.com, Version=v1
	case v1.SchemeGroupVersion.WithKind("FunctionIngress"):
		return &openfaasv1.FunctionIngressApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("FunctionIngressSpec"):
		return &openfaasv1.FunctionIngressSpecApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("FunctionIngressStatus"):
		return &openfaasv1.FunctionIngressStatusApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("FunctionIngressTLS"):
		return &openfaasv1.FunctionIngressTLSApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("ObjectReference"):
		return &openfaasv1.ObjectReferenceApplyConfiguration{}

	}
	return nil
}
