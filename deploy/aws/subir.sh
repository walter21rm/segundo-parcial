#!/usr/bin/env bash
# Publica las tres imágenes en ECR y aplica el overlay de EKS.
# Requiere: aws, docker, kubectl, eksctl ya usado para crear el clúster, y el AWS Load Balancer Controller instalado.
set -euo pipefail

REGION="${AWS_REGION:-us-east-1}"
CLUSTER="${CLUSTER_NAME:-feria}"
ACCOUNT="${AWS_ACCOUNT_ID:-}"
TAG="${IMAGE_TAG:-1.0.0}"

if [[ -z "$ACCOUNT" ]]; then
  ACCOUNT="$(aws sts get-caller-identity --query Account --output text)"
fi

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REGISTRY="${ACCOUNT}.dkr.ecr.${REGION}.amazonaws.com"
aws eks update-kubeconfig --region "$REGION" --name "$CLUSTER"

for nombre in feria-catalogo feria-pedidos feria-web; do
  aws ecr describe-repositories --repository-names "$nombre" --region "$REGION" >/dev/null 2>&1 \
    || aws ecr create-repository --repository-name "$nombre" --region "$REGION" >/dev/null
done

aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$REGISTRY"

docker build -t "$REGISTRY/feria-catalogo:$TAG" "$ROOT/servicios/catalogo"
docker build -t "$REGISTRY/feria-pedidos:$TAG" "$ROOT/servicios/pedidos"
docker build -t "$REGISTRY/feria-web:$TAG" "$ROOT/web"
docker push "$REGISTRY/feria-catalogo:$TAG"
docker push "$REGISTRY/feria-pedidos:$TAG"
docker push "$REGISTRY/feria-web:$TAG"

OVERLAY="$ROOT/deploy/k8s/overlays/aws"
(
  cd "$OVERLAY"
  kustomize edit set image \
    "feria/catalogo=$REGISTRY/feria-catalogo:$TAG" \
    "feria/pedidos=$REGISTRY/feria-pedidos:$TAG" \
    "feria/web=$REGISTRY/feria-web:$TAG"
)
kubectl apply -k "$OVERLAY"
kubectl -n feria rollout status deployment/catalogo
kubectl -n feria rollout status deployment/pedidos
kubectl -n feria rollout status deployment/web
echo "Ingress:"
kubectl -n feria get ingress web
