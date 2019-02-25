package resourceutil

import (
	"context"

	resourcesv1alpha1 "github.com/iljaweis/resource-ctlr/pkg/apis/resources/v1alpha1"
	pkgerrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Ready(c client.Client, namespace string, req *resourcesv1alpha1.Requires) (bool, error) {

	ready := true

	if req == nil {
		return true, nil
	}

	for _, r := range *req {
		thisready := false
		if r.Command != nil {
			x := &resourcesv1alpha1.Command{}
			err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: r.Command.Name}, x)
			if err != nil && !errors.IsNotFound(err) {
				return false, pkgerrors.Wrapf(err, "could not fetch command %s/%s", namespace, r.Command.Name)
			}
			if err == nil && x.Status.Done {
				thisready = true
			}
		}
		if r.File != nil {
			x := &resourcesv1alpha1.File{}
			err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: r.File.Name}, x)
			if err != nil && !errors.IsNotFound(err) {
				return false, pkgerrors.Wrapf(err, "could not fetch file %s/%s", namespace, r.File.Name)
			}
			if err == nil && x.Status.Done {
				thisready = true
			}
		}
		if r.FileContent != nil {
			x := &resourcesv1alpha1.FileContent{}
			err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: r.FileContent.Name}, x)
			if err != nil && !errors.IsNotFound(err) {
				return false, pkgerrors.Wrapf(err, "could not fetch filecontent %s/%s", namespace, r.FileContent.Name)
			}
			if err == nil && x.Status.Done {
				thisready = true
			}
		}
		if !thisready {
			ready = false
		}
	}

	return ready, nil
}
