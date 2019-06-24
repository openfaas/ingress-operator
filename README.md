OpenFaaS controller for FunctionIngress
====

WIP

```
rm ./artifacts/operator-amd64.yaml
kubectl apply -f ./artifacts/

go build && ./ingress-operator -kubeconfig=./config
```
