OpenFaaS controller for FunctionIngress
====

WIP

## Deployment

* In-cluster:

```sh
kubectl apply -f ./artifacts/

kubectl logs -n openfaas deploy/ingress-operator
```

* Local testing:

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