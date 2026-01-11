package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TodoListSpec defines the desired state of TodoList
type TodoListSpec struct {
	// Owner prefix for resource names and USER environment variable
	Owner string `json:"owner"`
}

// TodoListStatus defines the observed state of TodoList
type TodoListStatus struct {
	// Phase represents the current phase of the TodoList
	Phase string `json:"phase,omitempty"`

	// MariaDBReady indicates if MariaDB is ready
	MariaDBReady bool `json:"mariadbReady,omitempty"`

	// TodoAPIReady indicates if Todo API is ready
	TodoAPIReady bool `json:"todoApiReady,omitempty"`

	// FrontendReady indicates if Frontend is ready
	FrontendReady bool `json:"frontendReady,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.owner`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TodoList is the Schema for the todolists API
type TodoList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TodoListSpec   `json:"spec,omitempty"`
	Status TodoListStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TodoListList contains a list of TodoList
type TodoListList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TodoList `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TodoList{}, &TodoListList{})
}
