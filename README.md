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

This project addresses the following Proposal for Kubernetes: [Proposal: define custom hostname for functions #1082](https://github.com/openfaas/faas/issues/1082)

> Looking for a tutorial? See the [OpenFaaS documentation](https://docs.openfaas.com/reference/authentication/)

## Schema

This is an Operator / controller to build Kubernetes `Ingress` and JetStack `Certificate` objects for functions.

The following example would expose the `nodeinfo` function from the store as a URL: `nodeinfo.myfaas.club`.

```yaml
apiVersion: openfaas.com/v1alpha2
kind: FunctionIngress
metadata:
  name: nodeinfo
  namespace: openfaas
spec:
  domain: "nodeinfo.myfaas.club"
  function: "nodeinfo"
  ingressType: "nginx"
  # tls:
  #   enabled: true
  #   issuerRef:
  #     name: "letsencrypt-staging"
  #     kind: "Issuer"
```

Exploring the schema:

* The `domain` field corresponds to a DNS entry which points at your IngressController's public IP, or the IP of one of the hosts if using `HostPort`.

* `function` refers to the function you want to expose on the domain.

* `tls` whether to provision a TLS certificate using JetStack's [cert-manager](https://github.com/jetstack/cert-manager)

* `issuerRef` which issuer to use, this may be a staging or production issuer.

## Status

This is work-in-progress prototype and only suitable for development and testing. Contributions and suggestions are welcome.

Todo:
- [x] Create `Ingress` records for HTTP
- [x] Create `Ingress` records for HTTPS
- [x] Create cert-manager `Certificate` records
- [x] Support Nginx
- [x] Support Zoolando's Skipper
- [x] Support Traefik
- [x] Support armhf / Raspberry Pi
- [ ] Add `.travis.yml` for CI

## Deployment

### Pre-reqs

There are several pre-reqs for a working installation, but some of these components are installed with OpenFaaS and can also be found in [the docs](https://docs.openfaas.com/).

#### Install: `tiller`

```
curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get | bash

kubectl -n kube-system create sa tiller \
  && kubectl create clusterrolebinding tiller \
  --clusterrole cluster-admin \
  --serviceaccount=kube-system:tiller

## Wait for tiller
helm init --skip-refresh --upgrade --service-account tiller --wait
```

#### IngressController: `nginx`

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

#### OpenFaaS

OpenFaaS is also required:

```
git clone https://github.com/openfaas/faas-netes
cd faas-netes

kubectl apply -f namespaces.yml
# generate a random password
PASSWORD=$(head -c 12 /dev/urandom | shasum| cut -d' ' -f1)

kubectl -n openfaas create secret generic basic-auth \
--from-literal=basic-auth-user=admin \
--from-literal=basic-auth-password="$PASSWORD"

echo $PASSWORD > ../password.txt

kubectl apply -f ./yaml

kubectl port-forward -n openfaas deploy/gateway 31112:8080 &
echo -n ${PASSWORD} | faas-cli login --username admin --password-stdin -g 127.0.0.1:31112
faas-cli store deploy nodeinfo -g 127.0.0.1:31112
```

#### Configure DNS records

##### Find your public IP

Find the LB for Nginx:

```
kubectl get svc -n default
```

Or find the NodeIP:

```
kubectl get node -o wide
```

##### Create DNS A records

You should now configure your DNS A records:

For example, `nodeinfo` function in the `myfaas.club` domain and IP `178.128.137.209`:

```
nodeinfo.myfaas.club  178.128.137.209
```

> Note: with DigitalOcean's CLI you could run: `doctl compute domain create nodeinfo.myfaas.club --ip-address 178.128.137.209`.

##### TLS: Configure cert-manager

If using TLS, then install [cert-manager](https://docs.openfaas.com/reference/ssl/kubernetes-with-cert-manager/#install-cert-manager).

Now create an issuer:

```yaml
---
apiVersion: certmanager.k8s.io/v1alpha1
kind: Issuer
metadata:
  name: letsencrypt-staging
  namespace: openfaas
spec:
  acme:
    server: https://acme-staging.api.letsencrypt.org/directory
    # Email address used for ACME registration
    email: <your-email-here>
    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-staging
    http01: {}
```

Save as `letsencrypt-issuer.yaml` then run `kubectl apply -f letsencrypt-issuer.yaml`.

### Run or deploy the IngressOperator

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

### With TLS

```yaml
apiVersion: openfaas.com/v1alpha2
kind: FunctionIngress
metadata:
  name: nodeinfo-tls
  namespace: openfaas
spec:
  domain: "nodeinfo-tls.myfaas.club"
  function: "nodeinfo"
  ingressType: "nginx"
  tls:
    enabled: true
    issuerRef:
      name: "letsencrypt-staging"
      kind: "Issuer"
```

*nodeinfo.yaml*

### Without TLS

```yaml
apiVersion: openfaas.com/v1alpha2
kind: FunctionIngress
metadata:
  name: nodeinfo
  namespace: openfaas
spec:
  domain: "nodeinfo.myfaas.club"
  function: "nodeinfo"
  ingressType: "nginx"
```

*nodeinfo.yaml*

### Apply

```sh
kubectl apply -f nodeinfo.yaml
```

### Test:

```
# Find the ingress record
kubectl get ingress -n openfaas

# Find the cert record
kubectl get cert -n openfaas

# Find the FunctionIngress
kubectl get FunctionIngress -n openfaas
```

Remember to configure DNS for `nodeinfo.myfaas.club` or edit `/etc/hosts` and point to your `IngressController`'s IP or `LoadBalancer`.

## Contributing

This project follows the [OpenFaaS contributing guide](./CONTRIBUTING.md)

## LICENSE

MIT
