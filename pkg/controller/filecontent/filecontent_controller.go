package filecontent

import (
	"bytes"
	"context"
	"fmt"
	"github.com/iljaweis/resource-ctlr/pkg/resourceutil"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/types"
	"time"

	resourcesv1alpha1 "github.com/iljaweis/resource-ctlr/pkg/apis/resources/v1alpha1"
	hostc "github.com/iljaweis/resource-ctlr/pkg/controller/host"
	pkgerrors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_filecontent")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new FileContent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileFileContent{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("filecontent-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource FileContent
	err = c.Watch(&source.Kind{Type: &resourcesv1alpha1.FileContent{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileFileContent{}

// ReconcileFileContent reconciles a FileContent object
type ReconcileFileContent struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a FileContent object and makes changes based on the state read
// and what is in the FileContent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Stdout.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileFileContent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	logger.Info("Reconciling FileContent")

	ctx := context.TODO()

	fc := &resourcesv1alpha1.FileContent{}
	err := r.client.Get(ctx, request.NamespacedName, fc)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if fc.Status.Done {
		return reconcile.Result{}, nil // TODO: update?
	}

	ready, err := resourceutil.Ready(r.client, request.Namespace, fc.Spec.Requires)
	if err != nil {
		return reconcile.Result{}, pkgerrors.Wrap(err, "could not determine readiness")
	}

	if !ready {
		if fc.Status.StatusString != "WAITING" {
			fc.Status.StatusString = "WAITING"
			if err := r.client.Status().Update(ctx, fc); err != nil {
				return reconcile.Result{}, pkgerrors.Wrap(err, "could not update status")
			}
		}
		return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}

	host := &resourcesv1alpha1.Host{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: request.Namespace, Name: fc.Spec.Host}, host)
	if err != nil {
		if errors.IsNotFound(err) { // TODO: reschedule
			return reconcile.Result{},
				pkgerrors.Wrapf(err, "the host resource %s/%s does not exist", request.Namespace, fc.Spec.Host)
		} else {
			return reconcile.Result{},
				pkgerrors.Wrapf(err, "error fetching host %s/%s", request.Namespace, fc.Spec.Host)
		}
	}

	sshKeySecret := &corev1.Secret{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: request.Namespace, Name: host.Spec.SshKeySecret}, sshKeySecret)
	if err != nil {
		if errors.IsNotFound(err) { // TODO: reschedule
			return reconcile.Result{}, fmt.Errorf("secret %s/%s containing the private key for host %s was not found",
				request.Namespace, host.Spec.SshKeySecret, host.Name)
		} else {
			return reconcile.Result{}, pkgerrors.Wrapf(err,
				"error fetching secret %s/%s containing the private key for host %s",
				request.Namespace, host.Spec.SshKeySecret, host.Name)
		}
	}

	privateKey, err := ssh.ParsePrivateKey(sshKeySecret.Data[hostc.PrivateSshKeyField])
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("secret %s/%s does not have a field '%s' containing a valid private key",
			request.Namespace, host.Spec.SshKeySecret, hostc.PrivateSshKeyField)
	}

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshClient, err := ssh.Dial("tcp",
		fmt.Sprintf("%s:%d", host.Spec.IPAddress, host.Spec.Port),
		sshConfig)
	if err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "could not connect to host %s at root@%s:%d", host.Name, host.Spec.IPAddress, host.Spec.Port)
	}

	sshSession, err := sshClient.NewSession()
	if err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "could not connect to host %s at root@%s:%d", host.Name, host.Spec.IPAddress, host.Spec.Port)
	}

	var b bytes.Buffer
	sshSession.Stdout = &b

	if err := sshSession.Run(fmt.Sprintf("cat %s", fc.Spec.Path)); err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "could not run command %s on host %s", fc.Name, host.Name)
	}

	fc.Status.Done = true
	fc.Status.Content = b.String()
	fc.Status.StatusString = "DONE"

	if err := r.client.Status().Update(ctx, fc); err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "error storing results for command %s", fc.Name)
	}

	return reconcile.Result{}, nil
}
