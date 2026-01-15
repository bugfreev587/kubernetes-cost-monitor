#!/bin/bash
# Script to verify API key secret is configured correctly

SECRET_NAME="cost-agent-api-key"
API_KEY="17215fd3-a750-4ef2-a7d3-c5b58743fd2f:CeYC8PtLPiXBk4yVBU-IrNhR-RuXhb1m1EixYFFW5-o"

echo "Checking if secret exists..."
kubectl get secret $SECRET_NAME

echo ""
echo "Verifying secret value..."
SECRET_VALUE=$(kubectl get secret $SECRET_NAME -o jsonpath='{.data.api-key}' | base64 -d)
echo "Secret value (from Kubernetes): $SECRET_VALUE"
echo "Expected value: $API_KEY"

if [ "$SECRET_VALUE" = "$API_KEY" ]; then
    echo "✓ Secret value matches!"
else
    echo "✗ Secret value does NOT match!"
    echo ""
    echo "To update the secret, run:"
    echo "kubectl delete secret $SECRET_NAME"
    echo "kubectl create secret generic $SECRET_NAME --from-literal=api-key=\"$API_KEY\""
    echo "kubectl rollout restart deployment/cost-agent"
fi

echo ""
echo "Checking if pod can read the secret..."
POD_NAME=$(kubectl get pods -l app=cost-agent -o jsonpath='{.items[0].metadata.name}')
if [ -n "$POD_NAME" ]; then
    echo "Pod name: $POD_NAME"
    kubectl exec $POD_NAME -- env | grep AGENT_API_KEY
else
    echo "No pod found"
fi

