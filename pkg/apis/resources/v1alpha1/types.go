package v1alpha1

type Requires []Require

type Require struct {
	Command     *RequireCommand     `json:"command,omitempty"`
	File        *RequireFile        `json:"file,omitempty"`
	FileContent *RequireFileContent `json:"filecontent,omitempty"`
}

type RequireCommand struct {
	Name string `json:"name"`
}

type RequireFile struct {
	Name string `json:"name"`
}

type RequireFileContent struct {
	Name string `json:"name"`
}
