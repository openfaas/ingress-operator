OpenFaaS controller for FunctionIngress
====

## Why is this needed?

OpenFaaS functions are created as pairs with Deployment and Service objects, which eventually go on to create a Pod.

Deployments should not be exposed directly, but accessed via the OpenFaaS Gateway service.

The gateway has a number of rules including:
* providing HA through N replicas
* adding tracing IDs
* adding authz
* collecting metrics
* scaling endpoints from zero

Users started to create `Ingress` records pointing at the gateway for each public endpoint they wanted to host with a specific website address. This Operator automates that.

> See also: [Proposal: define custom hostname for functions #1082](https://github.com/openfaas/faas/issues/1082)

## Schema

This is an Operator / controller to build Kubernetes `Ingress` and JetStack `Certificate` objects for functions.

```yaml
apiVersion: openfaas.com/v1alpha2
kind: FunctionIngress
metadata:
  name: nodeinfo
  namespace: openfaas
spec:
  name: nodeinfo
  domain: nodeinfo.myfaas.club
  function: nodeinfo
  # tls: true # TBD
  # issuer: letsencrypt-prod #TDB
```

Exploring the schema:

* The `domain` field corresponds to a DNS entry which points at your IngressController's public IP, or the IP of one of the hosts if using `HostPort`.

* `function` refers to the function you want to expose on the domain.

* `tls` whether to provision a TLS certificate using JetStack's [cert-manager](https://github.com/jetstack/cert-manager)

* `issuer` which issuer to use, this may be a staging or production issuer.

## Status

This is work-in-progress prototype and only suitable for development and testing. Contributions and suggestions are welcome.

Todo:
- [x] Create `Ingress` records for HTTP
- [ ] Create `Ingress` records for HTTPS
- [ ] Create cert-manager `Certificate` records
- [ ] Add `.travis.yml` for CI

## Deployment

### Pre-reqs

[nginx IngressController](https://github.com/helm/charts/tree/master/stable/nginx-ingress) is recommended. Use a HostPort if testing against a local cluster where `LoadBalancer` is unavailable.

Make sure you have [helm](https://github.com/openfaas/faas-netes/blob/master/HELM.md) and Tiller.

Install [nginx](https://nginx.org/en/docs/) with LoadBalancer:

```sh
helm install stable/nginx-ingress --name nginxingress --set rbac.create=true
```

Install nginx with host-port:

```sh
export ADDITIONAL_SET=",controller.hostNetwork=true,controller.daemonset.useHostPort=true,dnsPolicy=ClusterFirstWithHostNet,controller.kind=DaemonSet"
helm install stable/nginx-ingress --name nginxingress --set rbac.create=true${ADDITIONAL_SET}
```

#### In-cluster:

```sh
kubectl apply -f ./artifacts/

kubectl logs -n openfaas deploy/ingress-operator
```

#### Local testing:

```sh
rm ./artifacts/operator-amd64.yaml
kubectl apply -f ./artifacts/

go build && ./ingress-operator -kubeconfig=./config
```

## Create your own `FunctionIngress`

```yaml
apiVersion: openfaas.com/v1alpha2
kind: FunctionIngress
metadata:
  name: nodeinfo
  namespace: openfaas
spec:
  name: nodeinfo
  domain: nodeinfo.myfaas.club
  function: nodeinfo
```

*nodeinfo.yaml*

```sh
kubectl apply -f nodeinfo.yaml
```

Now configure DNS for `nodeinfo.myfaas.club` or edit `/etc/hosts` and point to your `IngressController`'s IP or `LoadBalancer`.

## Contributing

This project follows the [OpenFaaS contributing guide](./CONTRIBUTING.md)

## LICENSE

MIT
