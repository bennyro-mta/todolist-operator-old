# TodoList Operator

A Kubernetes operator for managing TodoList applications with a Custom Resource Definition (CRD).

## Overview

The TodoList operator deploys a complete application stack including:
- **MariaDB**: Database backend
- **Todo API**: REST API service (`ghcr.io/bennyro-mta/todos-api:1.2`)
- **TodoList Vue**: Frontend UI (`ghcr.io/bennyro-mta/todolist-vue:1.2`)

### Key Features

- **Automatic Stack Deployment**: Single CRD creates entire application stack
- **Safe Spec Updates**: Change replicas/service type/API base URL after deployment
- **Immutable Owner**: `spec.owner` cannot be changed after creation (used as resource name prefix)
- **Multi-tenancy**: Use different owner prefixes for multiple isolated instances
- **Clean Resource Management**: All resources are cleaned up on deletion via owner references

## Prerequisites

- Kubernetes cluster (v1.28+) - minikube, kind, or any k8s cluster
- kubectl configured to access your cluster
- Docker (for building the operator image)
- Go 1.21+ (for local development)

## Quick Start

### 1. Install the CRDs

```bash
kubectl apply -f manifests/todolist-crd.yaml
# Or using make:
make install
```

### 2. Run the Operator

**Option A: Run Locally (Development)**
```bash
make deps
make run
# Or directly: go run main.go
```

**Option B: Deploy to Cluster (Production)**
```bash
# Build and push Docker image to your registry
make docker-build IMG=ghcr.io/bennyro-mta/todolist-operator:1.0
make docker-push IMG=ghcr.io/bennyro-mta/todolist-operator:1.0

# Update manifests/operator.yaml if using a different image/tag

# Deploy operator
make deploy
# Or: kubectl apply -f manifests/operator.yaml

# Verify operator is running
kubectl get pods -n todolist-operator-system
```

### 3. Create a TodoList Instance

```bash
# Using the sample
kubectl apply -f manifests/samples/todolist-sample.yaml

# Or create your own
cat <<EOF | kubectl apply -f -
apiVersion: todolist.example.com/v1
kind: TodoList
metadata:
  name: my-todolist
  namespace: default
spec:
  owner: john
EOF
```

This creates:
- `john-mariadb` - MariaDB deployment and service
- `john-todo-api` - Todo API deployment and service
- `john-todolist-vue` - Frontend deployment and service
- Associated ConfigMaps and Secrets

### 4. Verify Resources

```bash
kubectl get todolists
kubectl get deployments -l owner=john
kubectl get services -l owner=john
kubectl get pods -l owner=john
```

## Accessing the Application

### Port Forward to Frontend

```bash
kubectl port-forward deployment/john-todolist-vue 8080:8080
# Open browser to http://localhost:8080
```

### Port Forward to API

```bash
kubectl port-forward deployment/john-todo-api 8081:8080
curl http://localhost:8081/todos
```

## Updating Configuration

### Change `apiBaseUrl` (triggers Vue restart)

The frontend reads `API_BASE_URL` from a ConfigMap via `envFrom`. When you update `spec.apiBaseUrl`, the operator triggers a rolling restart:

```bash
kubectl patch todolist my-todolist --type merge -p '{"spec": {"apiBaseUrl": "/api/todos"}}'
kubectl rollout status deployment/john-todolist-vue
```

## Cleanup

```bash
# Delete a TodoList (cascades to all owned resources)
kubectl delete todolist my-todolist

# Undeploy operator
make undeploy

# Uninstall CRDs
make uninstall
```

## Troubleshooting

### Check Operator Logs

```bash
# If deployed to cluster
kubectl logs -n todolist-operator-system deployment/todolist-operator

# If running locally, check terminal output
```

### Check TodoList Status

```bash
kubectl describe todolist my-todolist
kubectl get pods -l owner=john
kubectl logs deployment/john-todo-api
kubectl logs deployment/john-mariadb
kubectl logs deployment/john-todolist-vue
```

### Common Issues

| Problem | Solution |
|---------|----------|
| TodoList stays in "Pending" phase | Check operator logs, verify images are accessible |
| Resources not created | Verify CRDs are installed, check RBAC permissions |

## API Reference

### TodoList Spec

| Field | Type | Description |
|-------|------|-------------|
| `owner` | string | Prefix for resource names and `USER` env var (required, immutable) |
| `frontendReplicas` | integer | Number of replicas for the frontend Deployment (default: 1) |
| `apiReplicas` | integer | Number of replicas for the API Deployment (default: 1) |
| `apiBaseUrl` | string | Base path the frontend uses to reach the API service (default: `/todos`); changing this triggers a Vue rollout |
| `serviceType` | string | Kubernetes Service type for frontend and API (default: `ClusterIP`) |

### TodoList Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase (Pending, Running, Failed) |
| `mariadbReady` | boolean | MariaDB deployment status |
| `todoApiReady` | boolean | Todo API deployment status |
| `frontendReady` | boolean | Frontend deployment status |

## Project Structure

```
todolist-operator/
├── api/v1/
│   ├── groupversion_info.go    # API group and version
│   └── todolist_types.go       # TodoList CRD types
├── controllers/
│   └── todolist_controller.go  # TodoList reconciler
├── manifests/
│   ├── todolist-crd.yaml       # TodoList CRD definition
│   ├── operator.yaml           # Operator deployment
│   └── samples/                # Sample TodoList resources
├── Dockerfile
├── go.mod
├── main.go
├── Makefile
└── README.md
```

## License

[MIT License](./LICENSE)
