package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	todolistv1 "todolist-operator/api/v1"
)

// TodoReconciler reconciles a Todo object
type TodoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// TaskRequest represents the API request/response structure
type TaskRequest struct {
	Task   string `json:"task"`
	Status string `json:"status,omitempty"`
}

// TaskResponse represents the API response structure
type TaskResponse struct {
	ID     int64  `json:"id"`
	Task   string `json:"task"`
	Status string `json:"status"`
}

// +kubebuilder:rbac:groups=todolist.example.com,resources=todos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=todolist.example.com,resources=todos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=todolist.example.com,resources=todos/finalizers,verbs=update
// +kubebuilder:rbac:groups=todolist.example.com,resources=todolists,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TodoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Todo instance
	todo := &todolistv1.Todo{}
	err := r.Get(ctx, req.NamespacedName, todo)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Todo resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Todo")
		return ctrl.Result{}, err
	}

	// Verify that the referenced TodoList exists
	todoListName := todo.Spec.TodoListName
	todoListInstance := &todolistv1.TodoList{}
	err = r.Get(ctx, client.ObjectKey{Name: todoListName, Namespace: todo.Namespace}, todoListInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "Referenced TodoList not found", "todoListName", todoListName)
			return ctrl.Result{RequeueAfter: time.Second * 30}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if TodoList is ready
	if todoListInstance.Status.Phase != "Running" {
		log.Info("TodoList is not running yet", "todoListName", todoListName, "phase", todoListInstance.Status.Phase)
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	owner := todoListInstance.Spec.Owner
	apiServiceURL := fmt.Sprintf("http://%s-todo-api.%s.svc.cluster.local:8080", owner, todo.Namespace)

	// If not yet synced, create in backend
	if !todo.Status.Synced {
		backendID, err := r.createTodoInBackend(ctx, apiServiceURL, todo)
		if err != nil {
			log.Error(err, "Failed to create todo in backend")
			return ctrl.Result{RequeueAfter: time.Second * 30}, err
		}

		// Update status
		now := metav1.Now()
		todo.Status.BackendID = backendID
		todo.Status.Synced = true
		todo.Status.LastSyncTime = &now
		if err := r.Status().Update(ctx, todo); err != nil {
			log.Error(err, "Failed to update Todo status")
			return ctrl.Result{}, err
		}

		log.Info("Successfully created todo in backend", "backendID", backendID)
		return ctrl.Result{}, nil
	}

	// If status changed, update in backend
	if err := r.updateTodoInBackend(ctx, apiServiceURL, todo); err != nil {
		log.Error(err, "Failed to update todo in backend")
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	// Update last sync time
	now := metav1.Now()
	todo.Status.LastSyncTime = &now
	if err := r.Status().Update(ctx, todo); err != nil {
		log.Error(err, "Failed to update Todo status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully synced todo with backend")
	return ctrl.Result{}, nil
}

func (r *TodoReconciler) createTodoInBackend(ctx context.Context, apiServiceURL string, todo *todolistv1.Todo) (int64, error) {
	taskReq := TaskRequest{
		Task:   todo.Spec.Task,
		Status: todo.Spec.Status,
	}

	jsonData, err := json.Marshal(taskReq)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal task request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiServiceURL+"/todos", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("backend API returned status %d: %s", resp.StatusCode, string(body))
	}

	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return taskResp.ID, nil
}

func (r *TodoReconciler) updateTodoInBackend(ctx context.Context, apiServiceURL string, todo *todolistv1.Todo) error {
	taskReq := TaskRequest{
		Status: todo.Spec.Status,
	}

	jsonData, err := json.Marshal(taskReq)
	if err != nil {
		return fmt.Errorf("failed to marshal task request: %w", err)
	}

	url := fmt.Sprintf("%s/todos/%d", apiServiceURL, todo.Status.BackendID)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backend API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TodoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&todolistv1.Todo{}).
		Complete(r)
}
