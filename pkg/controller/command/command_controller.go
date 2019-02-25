package command

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

var log = logf.Log.WithName("controller_command")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Command Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCommand{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("command-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Command
	err = c.Watch(&source.Kind{Type: &resourcesv1alpha1.Command{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCommand{}

// ReconcileCommand reconciles a Command object
type ReconcileCommand struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Command object and makes changes based on the state read
// and what is in the Command.Spec
func (r *ReconcileCommand) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	logger.Info("Reconciling Command")

	ctx := context.TODO()

	command := &resourcesv1alpha1.Command{}
	err := r.client.Get(ctx, request.NamespacedName, command)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if command.Status.Done {
		return reconcile.Result{}, nil
	}

	ready, err := resourceutil.Ready(r.client, request.Namespace, command.Spec.Requires)
	if err != nil {
		return reconcile.Result{}, pkgerrors.Wrap(err, "could not determine readiness")
	}

	if !ready {
		if command.Status.StatusString != "WAITING" {
			command.Status.StatusString = "WAITING"
			if err := r.client.Status().Update(ctx, command); err != nil {
				return reconcile.Result{}, pkgerrors.Wrap(err, "could not update status")
			}
		}
		return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}

	host := &resourcesv1alpha1.Host{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: request.Namespace, Name: command.Spec.Host}, host)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{},
				pkgerrors.Wrapf(err, "the host resource %s/%s does not exist", request.Namespace, command.Spec.Host)
		} else {
			return reconcile.Result{},
				pkgerrors.Wrapf(err, "error fetching host %s/%s", request.Namespace, command.Spec.Host)
		}
	}

	if command.Status.StatusString != "RUNNING" {
		command.Status.StatusString = "RUNNING"
		if err := r.client.Status().Update(ctx, command); err != nil {
			return reconcile.Result{}, pkgerrors.Wrap(err, "could not update status")
		}
	}

	sshKeySecret := &corev1.Secret{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: request.Namespace, Name: host.Spec.SshKeySecret}, sshKeySecret)
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

	var stdout, stderr bytes.Buffer
	sshSession.Stdout = &stdout
	sshSession.Stderr = &stderr

	var exitCode int

	err = sshSession.Run(command.Spec.Command)

	command.Status.Stdout = stdout.String()
	command.Status.Stderr = stderr.String()

	if err != nil {
		command.Status.StatusString = "FAILED"
		if exitError, ok := err.(*ssh.ExitError); ok {
			exitCode = exitError.ExitStatus()
		} else {
			return reconcile.Result{},
				pkgerrors.Wrapf(err, "could not run command %s on host %s", command.Name, host.Name)
		}
	} else {
		command.Status.StatusString = "DONE"
		command.Status.Done = true
		exitCode = 0
	}

	command.Status.ExitCode = exitCode

	if err := r.client.Status().Update(ctx, command); err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "error storing results for command %s", command.Name)
	}

	return reconcile.Result{}, nil
}
