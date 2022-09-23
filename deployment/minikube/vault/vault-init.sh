#!/bin/bash
# Setup: ./vault-init.sh --tls --helm

# check the dependencies for the script
. check-dependency.sh -n $@
set -e
set -x

#start vault server in dev mode.
vault server -dev -dev-root-token-id root -dev-listen-address 0.0.0.0:8200 & > /dev/null 2>&1
# only for minikube
kubectl config use-context minikube

# create namespace
kubectl apply -f vault-namespace.yml

# add the Hashicorp helm repository.
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update

has_param() {
    local term="$1"
    shift
    for arg; do
        if [[ $arg == "$term" ]]; then
            return 0
        fi
    done
    return 1
}

# generate tls certificate for vault agent.
function generate_tls_certificate(){
    openssl genrsa -out injector-ca.key 2048
    
    openssl req \
    -x509 \
    -new \
    -nodes \
    -key injector-ca.key \
    -sha256 \
    -days 1825 \
    -out injector-ca.crt \
    -subj "/C=US/ST=CA/L=San Francisco/O=HashiCorp/CN=vault-agent-injector-svc"
    
    openssl genrsa -out tls.key 2048
    
    
    openssl req \
    -new \
    -key tls.key \
    -out tls.csr \
    -subj "/C=US/ST=CA/L=San Francisco/O=HashiCorp/CN=vault-agent-injector-svc"
    
cat <<EOF >csr.conf
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = vault-agent-injector-svc
DNS.2 = vault-agent-injector-svc.vault
DNS.3 = vault-agent-injector-svc.vault.svc
DNS.4 = vault-agent-injector-svc.vault.svc.cluster.local
EOF
    
    
    openssl x509 \
    -req \
    -in tls.csr \
    -CA injector-ca.crt \
    -CAkey injector-ca.key \
    -CAcreateserial \
    -out tls.crt \
    -days 1825 \
    -sha256 \
    -extfile csr.conf
    
    # delete secret if it exists
    kubectl delete secret injector-tls --namespace=vault || true
    kubectl create secret generic injector-tls \
    --from-file tls.crt \
    --from-file tls.key \
    --namespace=vault
    
}
# install the Vault Helm chart with external Vault.
function helm_install_vault(){
    export CA_BUNDLE=$(cat injector-ca.crt | base64)
    export PRIVATE_IP=$(ipconfig getifaddr en0)
    helm uninstall vault -n vault || true
    helm upgrade --install --wait vault hashicorp/vault \
    --namespace=vault \
    --set="injector.logLevel=debug" \
    --set="injector.logFormat=json" \
    --set="injector.externalVaultAddr=http://local-vault.lambdatest.io:8200" \
    --set="injector.certs.secretName=injector-tls" \
    --set="injector.certs.caBundle=${CA_BUNDLE?}" \
    --version 0.19.0
}

if has_param '--tls' "$@"; then
    generate_tls_certificate
fi

if has_param '--helm' "$@"; then
    helm_install_vault
fi

export VAULT_ADDR=http://0.0.0.0:8200
vault login root

# enable Vault k8s auth method
vault auth enable kubernetes || true

VAULT_HELM_SECRET_NAME=$(kubectl get secrets --output=json -n vault | jq -r '.items[].metadata | select(.name|startswith("vault-token-")).name')

kubectl describe secret $VAULT_HELM_SECRET_NAME -n vault
TOKEN_REVIEW_JWT=$(kubectl get secret $VAULT_HELM_SECRET_NAME -n vault --output='go-template={{ .data.token }}' | base64 --decode)
KUBE_CA_CERT=$(kubectl config view --raw --minify --flatten --output='jsonpath={.clusters[].cluster.certificate-authority-data}' | base64 --decode)
KUBE_HOST=$(kubectl config view --raw --minify --flatten --output='jsonpath={.clusters[].cluster.server}')


currentver=$(kubectl version --short -o json | jq .serverVersion.gitVersion)
requiredver="\"v1.21.0\""

#Tell Vault how to communicate with the K8s (Minikube) cluster.
if [ "$(printf '%s\n' "$requiredver" "$currentver" | sort -V | head -n1)" = "$requiredver" ]; then
    vault write auth/kubernetes/config \
    token_reviewer_jwt="$TOKEN_REVIEW_JWT" \
    kubernetes_host="$KUBE_HOST" \
    kubernetes_ca_cert="$KUBE_CA_CERT" \
    issuer="https://kubernetes.default.svc.cluster.local"
else
    vault write auth/kubernetes/config \
    token_reviewer_jwt="$TOKEN_REVIEW_JWT" \
    kubernetes_host="$KUBE_HOST" \
    kubernetes_ca_cert="$KUBE_CA_CERT"
fi


MOUNT_ACCESSOR=$(vault auth list -format=json | jq -r '."kubernetes/".accessor')

# create Vault policy for accessing user secrets.
vault policy write kubernetes-kv-read - << EOF
    path "secret/data/github/{{identity.entity.aliases.$MOUNT_ACCESSOR.metadata.service_account_namespace}}/{{identity.entity.aliases.$MOUNT_ACCESSOR.metadata.service_account_name}}/*" {
        capabilities=["read"]
    }
    path "secret/data/gitlab/{{identity.entity.aliases.$MOUNT_ACCESSOR.metadata.service_account_namespace}}/{{identity.entity.aliases.$MOUNT_ACCESSOR.metadata.service_account_name}}/*" {
        capabilities=["read"]
    }
    path "secret/data/bitbucket/{{identity.entity.aliases.$MOUNT_ACCESSOR.metadata.service_account_namespace}}/{{identity.entity.aliases.$MOUNT_ACCESSOR.metadata.service_account_name}}/*" {
        capabilities=["read"]
    }
EOF

# create secrets engine for microservices
vault secrets enable -version=2  --path=microservices kv

# create policy for neuron
vault policy write neuron - << EOF
# Manage user secret data
path "secret/data/*" {
  capabilities = ["read","create","update","delete","list"]
}

# Need access to delete metadata
path "secret/metadata/*" {
  capabilities = ["delete"]
}

# Read the config
path "microservices/data/phoenix/neuron" {
  capabilities = ["read","list"]
}

# Manage kubernetes auth method
path "auth/kubernetes/role/*"
{
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF

#Create a role phoenix-neuron, example, that maps the Kubernetes Service Account to Vault policies and default token TTL.
vault write auth/kubernetes/role/phoenix-neuron \
bound_service_account_names=neuron \
bound_service_account_namespaces=phoenix \
policies=neuron

# Create policy for other microservices. (each microservice will have a namespace and seperate service account)
vault policy write microservices-kv-read - << EOF
    path "microservices/data/{{identity.entity.aliases.$MOUNT_ACCESSOR.metadata.service_account_namespace}}/{{identity.entity.aliases.$MOUNT_ACCESSOR.metadata.service_account_name}}" {
        capabilities=["read"]
    }
EOF


# put neuron config file in vault
vault kv put microservices/phoenix/neuron @../../../.nu.json



# openssl genrsa  -out private.pem 2048
# openssl rsa -in private.pem -outform PEM -pubout -out public.pem
# private_key=$(cat private.pem | base64)
# public_key=$(cat public.pem | base64)


# contents=$(jq --arg priv "$private_key" --arg pub "$public_key" \
# '. + {("jwt"): {"privateKey": $priv, "publicKey": $pub, "timeout": 2592000000000000}}' .nu.json)

# echo "${contents}" > .nu.json

# vault kv put microservices/phoenix/neuron @.nu.json