package controller

import (
	"github.com/iljaweis/resource-ctlr/pkg/controller/filecontents"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, filecontents.Add)
}
