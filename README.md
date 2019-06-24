OpenFaaS controller for FunctionIngress
====

WIP

Local testing:

```
rm ./artifacts/operator-amd64.yaml
kubectl apply -f ./artifacts/

go build && ./ingress-operator -kubeconfig=./config
```

In-cluster:


Local testing:

```
kubectl apply -f ./artifacts/

kubectl logs -n openfaas deploy/ingress-operator
```
