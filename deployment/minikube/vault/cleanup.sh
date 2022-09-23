#!/bin/bash

# only for minikube
kubectl config use-context minikube

helm uninstall vault -n vault
if [ "$(kubectl get namespace | grep -c vault)" != "0" ]; then
	kubectl delete namespace vault --wait
fi
pid_to_kill=$(lsof -i tcp:8200 | awk '/vault/ {print $2}' | head -1)
if [[ ! -z "$pid_to_kill" ]]; then
	kill -9 $pid_to_kill
fi
