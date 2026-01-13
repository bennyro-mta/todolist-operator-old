# Quick Start Guide

This guide will help you quickly deploy and test the TodoList Operator.

## Prerequisites

- Kubernetes cluster (minikube, kind, or any k8s cluster)
- kubectl configured
- Docker (for building the image)

## Quick Deploy

### Step 1: Install the CRDs

```bash
kubectl apply -f manifests/todolist-crd.yaml
```

### Step 2: Build the Operator

```bash
# Using Makefile
make deps
make build

# Or manually
go mod download
go mod tidy
go build -o manager main.go
```

### Step 3: Run Locally (Development)

For testing, you can run the operator locally:

```bash
# Make sure CRDs are installed first
make run

# Or directly
go run main.go
```

### Step 4: Deploy to Cluster (Production)

```bash
# Build Docker image
make docker-build IMG=todolist-operator:latest

# If using local cluster (minikube/kind), load the image
# For minikube:
minikube image load todolist-operator:latest

# For kind:
kind load docker-image todolist-operator:latest

# Deploy operator
kubectl apply -f manifests/operator.yaml
kubectl apply -f manifests/rbac.yaml

# Check operator is running
kubectl get pods -n todolist-operator-system
```

## Create a TodoList Instance

```bash
# Create a sample TodoList
kubectl apply -f manifests/samples/todolist-sample.yaml

# Check the TodoList status
kubectl get todolists
kubectl get todolists demo-todolist -o yaml

# Check created resources
kubectl get deployments -l owner=demo
kubectl get services -l owner=demo
kubectl get pods -l owner=demo
```

You should see:
- `demo-mariadb` deployment and service
- `demo-todo-api` deployment and service
- `demo-todolist-vue` deployment and service

## Change `apiBaseUrl` (triggers Vue restart)

The frontend reads `API_BASE_URL` from a ConfigMap via `envFrom`, which only takes effect at container start.
When you update `spec.apiBaseUrl`, the operator triggers a rolling restart of the `demo-todolist-vue` Deployment.

```bash
# Change the API base URL
kubectl patch todolist demo-todolist --type merge -p '{"spec": {"apiBaseUrl": "/todos"}}'

# Verify a rollout happens
kubectl rollout status deployment/demo-todolist-vue

# Optional: verify the pod-template annotation matches the desired value
kubectl get deploy demo-todolist-vue -o jsonpath='{.spec.template.metadata.annotations.todolist\.example\.com/api-base-url}{"\n"}'
```

## Access the Application

```bash
# Edit a todo
kubectl edit todo sample-todo-3

# Change spec.status from "pending" to "completed"
# Save and exit

# Verify it was synced
kubectl get todo sample-todo-3 -o yaml | grep -A 5 status
```

## Access the Application

### Port Forward to Frontend

```bash
# Get the frontend pod name
kubectl get pods -l app=todolist-vue,owner=demo

# Port forward
kubectl port-forward deployment/demo-todolist-vue 8080:8080

# Open browser to http://localhost:8080
```

### Port Forward to API

```bash
# Port forward to API
kubectl port-forward deployment/demo-todo-api 8081:8080

# Test API
curl http://localhost:8081/todos
```

## Cleanup

```bash
# Delete todolist (this will cascade delete all resources)
kubectl delete -f manifests/samples/todolist-sample.yaml

# Undeploy operator
make undeploy

# Uninstall CRDs
make uninstall
```

## Troubleshooting

### Check Operator Logs

```bash
# If running locally
# Check terminal output

# If deployed to cluster
kubectl logs -n todolist-operator-system deployment/todolist-operator
```

### Check TodoList Status

```bash
kubectl describe todolist demo-todolist
```

### Check Pod Status

```bash
# Check all pods for a TodoList
kubectl get pods -l owner=demo

# Check specific pod logs
kubectl logs deployment/demo-todo-api
kubectl logs deployment/demo-mariadb
kubectl logs deployment/demo-todolist-vue
```

### Common Issues

**TodoList stays in "Pending" phase:**
- Check operator logs
- Verify image pull policy and images are accessible
- Check RBAC permissions

**Resources not created:**
- Check operator has correct RBAC permissions
- Verify CRDs are installed
- Check operator logs for errors

## Testing Multiple Instances

You can create multiple TodoList instances with different owners:

```yaml
# todolist-john.yaml
apiVersion: todolist.example.com/v1
kind: TodoList
metadata:
  name: john-todolist
  namespace: default
spec:
  owner: john
---
# todolist-jane.yaml
apiVersion: todolist.example.com/v1
kind: TodoList
metadata:
  name: jane-todolist
  namespace: default
spec:
  owner: jane
```

Each will create isolated resources with their respective owner prefixes.
