package host

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/types"

	resourcesv1alpha1 "github.com/iljaweis/resource-ctlr/pkg/apis/resources/v1alpha1"
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

const (
	PrivateSshKeyField string = "ssh-privatekey"

	FactUname string = "uname"
)

var log = logf.Log.WithName("controller_host")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Host Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileHost{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("host-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Host
	err = c.Watch(&source.Kind{Type: &resourcesv1alpha1.Host{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileHost{}

// ReconcileHost reconciles a Host object
type ReconcileHost struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Host object and makes changes based on the state read
// and what is in the Host.Spec
func (r *ReconcileHost) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Host")

	host := &resourcesv1alpha1.Host{}
	err := r.client.Get(context.TODO(), request.NamespacedName, host)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if host.Status.Ready {
		// Do nothing.
		return reconcile.Result{}, nil
	}

	sshKeySecret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: request.Namespace, Name: host.Spec.SshKeySecret}, sshKeySecret)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, fmt.Errorf("secret %s/%s containing the private key for host %s was not found",
				request.Namespace, host.Spec.SshKeySecret, host.Name)
		} else {
			return reconcile.Result{}, pkgerrors.Wrapf(err,
				"error fetching secret %s/%s containing the private key for host %s",
				request.Namespace, host.Spec.SshKeySecret, host.Name)
		}
	}

	privateKey, err := ssh.ParsePrivateKey(sshKeySecret.Data[PrivateSshKeyField])
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("secret %s/%s does not have a field '%s' containing a valid private key",
			request.Namespace, host.Spec.SshKeySecret, PrivateSshKeyField)
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

	if err := sshSession.Run("/usr/bin/uname -a"); err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "could not check status of host %s at root@%s:%d", host.Name, host.Spec.IPAddress, host.Spec.Port)
	}

	host.Status.Ready = true
	if host.Status.Facts == nil {
		host.Status.Facts = make(map[string]string)
	}
	host.Status.Facts[FactUname] = b.String()

	if err := r.client.Status().Update(context.TODO(), host); err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "error saving ready state for host %s", host.Name)
	}

	return reconcile.Result{}, nil
}
