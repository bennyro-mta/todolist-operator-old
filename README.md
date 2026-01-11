# TodoList Operator

A Kubernetes operator for managing TodoList applications with two Custom Resource Definitions (CRDs):

- **TodoList**: Deploys a complete todolist application stack including MariaDB, Todo API, and Vue.js frontend
- **Todo**: Manages individual todo items that sync with the Todo API backend

## Architecture

The operator manages the following components:

### TodoList CRD
- Creates a complete todolist application stack
- Components:
  - **MariaDB**: Database backend
  - **Todo API**: REST API service (`ghcr.io/bennyro-mta/todos-api:1.2`)
  - **TodoList Vue**: Frontend UI (`ghcr.io/bennyro-mta/todolist-vue:1.2`)
- Uses `owner` field as prefix for all resource names
- Sets `USER` environment variable in frontend to the owner value
- Immutable after deployment (changes to spec are not reconciled)

### Todo CRD
- Creates individual todo items
- Syncs with the Todo API backend service
- Only the `status` field is mutable after creation
- Requires a running TodoList instance

## Installation

### Prerequisites
- Kubernetes cluster (v1.28+)
- kubectl configured to access your cluster
- Docker (for building the operator image)

### Deploy the Operator

1. **Apply the CRDs**:
```bash
kubectl apply -f manifests/todolist-crd.yaml
```

2. **Create the operator namespace and RBAC**:
```bash
kubectl apply -f manifests/operator.yaml
kubectl apply -f manifests/rbac.yaml
```

3. **Build and push the operator image** (optional - if using your own registry):
```bash
# Build the image
docker build -t todolist-operator:latest .

# Tag for your registry
docker tag todolist-operator:latest your-registry/todolist-operator:latest

# Push to your registry
docker push your-registry/todolist-operator:latest

# Update the image in manifests/operator.yaml
```

4. **Deploy the operator**:
```bash
kubectl apply -f manifests/operator.yaml
```

5. **Verify the operator is running**:
```bash
kubectl get pods -n todolist-operator-system
```

## Usage

### Create a TodoList Instance

Create a file `my-todolist.yaml`:

```yaml
apiVersion: todolist.example.com/v1
kind: TodoList
metadata:
  name: my-todolist
  namespace: default
spec:
  owner: john
```

Apply it:
```bash
kubectl apply -f my-todolist.yaml
```

This will create:
- `john-mariadb` - MariaDB deployment and service
- `john-todo-api` - Todo API deployment and service
- `john-todolist-vue` - Frontend deployment and service
- Associated ConfigMaps and Secrets

Check the status:
```bash
kubectl get todolists
kubectl get deployments -l owner=john
kubectl get services -l owner=john
```

### Create Todo Items

Once the TodoList is running, create todos:

Create a file `my-todo.yaml`:

```yaml
apiVersion: todolist.example.com/v1
kind: Todo
metadata:
  name: my-first-todo
  namespace: default
spec:
  todolistName: my-todolist
  task: "Buy groceries"
  status: "pending"
```

Apply it:
```bash
kubectl apply -f my-todo.yaml
```

### Update Todo Status

Edit the status field:
```bash
kubectl edit todo my-first-todo
```

Change the `spec.status` field from "pending" to "completed". The operator will sync this change to the backend API.

### List Todos

```bash
kubectl get todos
```

## Development

### Build the Operator

```bash
# Download dependencies
go mod download
go mod tidy

# Build locally
go build -o manager main.go

# Run locally (requires kubeconfig)
./manager
```

### Testing

```bash
# Install CRDs
kubectl apply -f manifests/todolist-crd.yaml

# Run the operator locally
go run main.go
```

## Project Structure

```
todolist-operator/
├── api/
│   └── v1/
│       ├── groupversion_info.go    # API group and version
│       └── todolist_types.go       # TodoList CRD types
├── controllers/
│   └── todolist_controller.go      # TodoList reconciler
├── manifests/
│   ├── todolist-crd.yaml           # TodoList CRD definition
│   ├── rbac.yaml                   # RBAC configuration
│   └── operator.yaml               # Operator deployment
├── Dockerfile                      # Container image definition
├── go.mod                          # Go module definition
├── main.go                         # Operator entry point
└── README.md                       # This file
```

## Features

- **Automatic Stack Deployment**: Single CRD creates entire application stack
- **Immutable TodoList**: Once deployed, TodoList specs cannot be changed
- **Multi-tenancy**: Use different owner prefixes for multiple instances
- **Clean Resource Management**: All resources owned by CRDs are cleaned up on deletion

## API Reference

### TodoList Spec

| Field | Type | Description |
|-------|------|-------------|
| `owner` | string | Prefix for resource names and USER env var (required) |

### TodoList Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase (Pending, Running, Failed) |
| `mariadbReady` | boolean | MariaDB deployment status |
| `todoApiReady` | boolean | Todo API deployment status |
| `frontendReady` | boolean | Frontend deployment status |

### Todo Spec

| Field | Type | Description |
|-------|------|-------------|
| `todolistName` | string | Name of TodoList instance (required) |
| `task` | string | Task description (required) |
| `status` | string | Task status (required) |

### Todo Status

| Field | Type | Description |
|-------|------|-------------|
| `backendId` | int64 | ID from backend API |
| `synced` | boolean | Sync status with backend |
| `lastSyncTime` | timestamp | Last sync timestamp |

## License

This project follows the same license as the todolist application.

## Notes

- The operator uses controller-runtime framework without operator-sdk
- Resources are created with owner references for automatic cleanup
- Todo status updates trigger backend API calls
- TodoList deployments use images from ghcr.io/bennyro-mta
