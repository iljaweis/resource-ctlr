package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FileContentSpec defines the desired state of FileContent
type FileContentSpec struct {
	Host     string    `json:"host"`
	Path     string    `json:"path"`
	Requires *Requires `json:"requires,omitempty"`
}

// FileContentStatus defines the observed state of FileContent
type FileContentStatus struct {
	Done         bool   `json:"done"`
	Content      string `json:"content"`
	StatusString string `json:"status_string"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FileContent is the Schema for the filecontents API
// +k8s:openapi-gen=true
type FileContent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FileContentSpec   `json:"spec,omitempty"`
	Status FileContentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FileContentList contains a list of FileContent
type FileContentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FileContent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FileContent{}, &FileContentList{})
}
