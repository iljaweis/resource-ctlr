package file

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/iljaweis/resource-ctlr/pkg/resourceutil"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/types"

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

var log = logf.Log.WithName("controller_file")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new File Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileFile{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("file-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource File
	err = c.Watch(&source.Kind{Type: &resourcesv1alpha1.File{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileFile{}

// ReconcileFile reconciles a File object
type ReconcileFile struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a File object and makes changes based on the state read
// and what is in the File.Spec
func (r *ReconcileFile) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	logger.Info("Reconciling FileContent")

	ctx := context.TODO()

	f := &resourcesv1alpha1.File{}
	err := r.client.Get(ctx, request.NamespacedName, f)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if f.Status.Done {
		return reconcile.Result{}, nil // TODO: update?
	}

	ready, err := resourceutil.Ready(r.client, request.Namespace, f.Spec.Requires)
	if err != nil {
		return reconcile.Result{}, pkgerrors.Wrap(err, "could not determine readiness")
	}

	if !ready {
		if f.Status.StatusString != "WAITING" {
			f.Status.StatusString = "WAITING"
			if err := r.client.Status().Update(ctx, f); err != nil {
				return reconcile.Result{}, pkgerrors.Wrap(err, "could not update status")
			}
		}
		return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}

	content, contentReady, err := r.getContent(f)
	if err != nil {
		return reconcile.Result{}, pkgerrors.Wrapf(err, "could not fetch content for %s does not exist", request)
	}

	if !contentReady {
		return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}

	host := &resourcesv1alpha1.Host{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: request.Namespace, Name: f.Spec.Host}, host)
	if err != nil {
		if errors.IsNotFound(err) { // TODO: reschedule
			return reconcile.Result{},
				pkgerrors.Wrapf(err, "the host resource %s/%s does not exist", request.Namespace, f.Spec.Host)
		} else {
			return reconcile.Result{},
				pkgerrors.Wrapf(err, "error fetching host %s/%s", request.Namespace, f.Spec.Host)
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

	// what could possibly go wrong
	if err := sshSession.Run(fmt.Sprintf("cat > %s <<'EOF'\n%s\nEOF", f.Spec.Path, content)); err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "could not run command %s on host %s", f.Name, host.Name)
	}

	f.Status.Done = true
	f.Status.StatusString = "DONE"

	if err := r.client.Status().Update(ctx, f); err != nil {
		return reconcile.Result{},
			pkgerrors.Wrapf(err, "error storing results for command %s", f.Name)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileFile) getContent(f *resourcesv1alpha1.File) (string, bool, error) {

	if f.Spec.Content != "" {
		return f.Spec.Content, true, nil
	}
	if f.Spec.Source == nil {
		return "", false, nil
	}

	if f.Spec.Source.FileContent != nil {
		fmt.Printf("getting filecontent %s\n", f.Spec.Source.FileContent.Name)
		fc := &resourcesv1alpha1.FileContent{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: f.Namespace, Name: f.Spec.Source.FileContent.Name}, fc)
		if err != nil && !errors.IsNotFound(err) {
			return "", false, pkgerrors.Wrapf(err, "could not get filecontent %s/%s", f.Namespace, f.Spec.Source.FileContent.Name)
		}
		if err != nil {
			return "", false, nil
		}
		if fc.Status.Done {
			return fc.Status.Content, true, nil
		} else {
			return "", false, nil
		}
	}

	return "", false, nil
}
