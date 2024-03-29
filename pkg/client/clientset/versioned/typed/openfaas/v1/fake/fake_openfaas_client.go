/*
Copyright 2023 OpenFaaS Author(s)

Licensed under the MIT license. See LICENSE file in the project root for full license information.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/openfaas/ingress-operator/pkg/client/clientset/versioned/typed/openfaas/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeOpenfaasV1 struct {
	*testing.Fake
}

func (c *FakeOpenfaasV1) FunctionIngresses(namespace string) v1.FunctionIngressInterface {
	return &FakeFunctionIngresses{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeOpenfaasV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
