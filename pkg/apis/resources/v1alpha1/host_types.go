package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HostSpec defines the desired state of Host
type HostSpec struct {
	SshKeySecret string `json:"sshkeysecret"`
	IPAddress    string `json:"ipaddress"`
	Port         int    `json:"port"`
}

// HostStatus defines the observed state of Host
type HostStatus struct {
	Ready bool              `json:"ready"`
	Facts map[string]string `json:"facts"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Host is the Schema for the hosts API
// +k8s:openapi-gen=true
type Host struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostSpec   `json:"spec,omitempty"`
	Status HostStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostList contains a list of Host
type HostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Host `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Host{}, &HostList{})
}
