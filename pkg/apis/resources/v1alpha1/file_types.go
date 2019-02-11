package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FileSpec defines the desired state of File
type FileSpec struct {
	Host     string      `json:"host"`
	Path     string      `json:"path"`
	Content  string      `json:"content,omitempty"`
	Source   *FileSource `json:"source,omitempty"`
	Requires *Requires   `json:"requires,omitempty"`
}

type FileSource struct {
	FileContent   *FileSourceFileContent   `json:"filecontent,omitempty"`
	CommandOutput *FileSourceCommandOutput `json:"commandoutput,omitempty"`
	ConfigMap     *FileSourceConfigMap     `json:"configmap,omitempty"`
	Secret        *FileSourceSecret        `json:"configmap,omitempty"`
}

type FileSourceFileContent struct {
	Name string `json:"name"`
}

type FileSourceCommandOutput struct {
	Name string `json:"name"`
}

type FileSourceConfigMap struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type FileSourceSecret struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// FileStatus defines the observed state of File
type FileStatus struct {
	Done bool `json:"done"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// File is the Schema for the files API
// +k8s:openapi-gen=true
type File struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FileSpec   `json:"spec,omitempty"`
	Status FileStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FileList contains a list of File
type FileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []File `json:"items"`
}

func init() {
	SchemeBuilder.Register(&File{}, &FileList{})
}
