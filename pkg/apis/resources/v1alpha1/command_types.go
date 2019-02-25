package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CommandSpec defines the desired state of Command
type CommandSpec struct {
	Host     string    `json:"host"`
	Command  string    `json:"command"`
	Requires *Requires `json:"requires,omitempty"`
}

// CommandStatus defines the observed state of Command
type CommandStatus struct {
	Done         bool   `json:"done"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	ExitCode     int    `json:"exitcode"`
	StatusString string `json:"status_string"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Command is the Schema for the commands API
// +k8s:openapi-gen=true
type Command struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CommandSpec   `json:"spec,omitempty"`
	Status CommandStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CommandList contains a list of Command
type CommandList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Command `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Command{}, &CommandList{})
}
