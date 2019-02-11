package controller

import (
	"github.com/iljaweis/resource-ctlr/pkg/controller/filecontent"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, filecontent.Add)
}
