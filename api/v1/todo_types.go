package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TodoSpec defines the desired state of Todo
type TodoSpec struct {
	// TodoListName is the name of the TodoList instance this todo belongs to
	TodoListName string `json:"todolistName"`

	// Task is the task description
	Task string `json:"task"`

	// Status is the task status (pending, completed, etc.)
	Status string `json:"status"`
}

// TodoStatus defines the observed state of Todo
type TodoStatus struct {
	// BackendID is the ID from the backend todo-api service
	BackendID int64 `json:"backendId,omitempty"`

	// Synced indicates whether the todo is synced with the backend
	Synced bool `json:"synced,omitempty"`

	// LastSyncTime is the last time the todo was synced with backend
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Task",type=string,JSONPath=`.spec.task`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.spec.status`
// +kubebuilder:printcolumn:name="TodoList",type=string,JSONPath=`.spec.todolistName`
// +kubebuilder:printcolumn:name="Synced",type=boolean,JSONPath=`.status.synced`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Todo is the Schema for the todos API
type Todo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TodoSpec   `json:"spec,omitempty"`
	Status TodoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TodoItemList contains a list of Todo
type TodoItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Todo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Todo{}, &TodoItemList{})
}

// DeepCopyObject implements runtime.Object for Todo
func (in *Todo) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(Todo)
	*out = *in
	return out
}

// DeepCopyObject implements runtime.Object for TodoItemList
func (in *TodoItemList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(TodoItemList)
	*out = *in
	if in.Items != nil {
		out.Items = make([]Todo, len(in.Items))
		copy(out.Items, in.Items)
	}
	return out
}
