package tool

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
)

const (
	ConditionStatusBuildFailed   = "BuildFailed"
	ConditionStatusReferFailed   = "ReferFailed"
	ConditionStatusReferSuccess  = "ReferSuccess"
	ConditionStatusCreateFailed  = "CreateFailed"
	ConditionStatusCreateSuccess = "CreateSuccess"
	ConditionStatusAlreadyExists = "AlreadyExists"
	ConditionStatusFetchFailed   = "FetchFailed"
	ConditionStatusUpdateFailed  = "UpdateFailed"
	ConditionStatusUpdateSuccess = "UpdateSuccess"

	RetryNever = 0 * time.Second
	RetryLater = 10 * time.Second
)

type BasicReconciler struct {
	Log       logr.Logger
	Client    client.Client
	ApiReader client.Reader
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
}

func NewBasicReconciler(mgr manager.Manager, resourceType string) BasicReconciler {
	return BasicReconciler{
		Log:       logf.Log.WithName(resourceType),
		Scheme:    mgr.GetScheme(),
		Client:    mgr.GetClient(),
		ApiReader: mgr.GetAPIReader(),
		Recorder:  mgr.GetEventRecorderFor(resourceType),
	}
}

func (r BasicReconciler) ResourceLogger(resource client.Object, args ...interface{}) logr.Logger {
	meta := ParseResourceMeta(resource, args...)
	return r.Log.WithValues(meta.Kind, meta.NamespacedName)
}

// FetchResource return notFound, error
func (r BasicReconciler) FetchResource(ctx context.Context, resource client.Object, args ...interface{}) (bool, error) {
	name := ParseNamespacedName(resource, args...)
	if err := r.Client.Get(ctx, name, resource); err != nil {
		return errors.IsNotFound(err), err
	}
	return false, nil
}

// NewSetupTemplate resource build -> create -> update -> update status
func (r BasicReconciler) NewSetupTemplate(ctx context.Context, callbacks SetupCallbacks) *SetupTemplate {
	// check context
	if ctx == nil {
		ctx = context.Background()
	}
	return &SetupTemplate{
		Context:         ctx,
		BasicReconciler: r,
		callbacks:       &callbacks,
	}
}

// NewCreateTemplate resource build -> create
func (r BasicReconciler) NewCreateTemplate(ctx context.Context, ownerReferred bool, builder BuildResource) *SetupTemplate {
	return r.NewSetupTemplate(ctx, SetupCallbacks{
		NotOwnerReferred: !ownerReferred,
		BuildResource:    builder,
	})
}

// NewUpdateTemplate resource update
func (r BasicReconciler) NewUpdateTemplate(ctx context.Context, updater UpdateResource) *SetupTemplate {
	return r.NewSetupTemplate(ctx, SetupCallbacks{
		UpdateResource: updater,
	})
}

// BuildResource create the resource owned by owner
type BuildResource func(sc SetupContext, owner client.Object, name string, params ...interface{}) (client.Object, error)

// UpdateResource according to owner definitions,
// modification of owner status will automatically be saved,
// return true if resource been changed and need to update,
// cannot always return true !!!
type UpdateResource func(sc SetupContext, owner client.Object, resource client.Object, params ...interface{}) (bool, error)

// UpdateOwnerStatus updates owner's status subresource,
// cannot always return true !!!
type UpdateOwnerStatus func(sc SetupContext, owner client.Object, resource client.Object, params ...interface{}) (bool, error)

type SetupCallbacks struct {
	NotOwnerReferred bool

	BuildResource     BuildResource
	UpdateResource    UpdateResource
	UpdateOwnerStatus UpdateOwnerStatus
}

// SetupTemplate a resource setup framework
// 1. fetch resource
// 2.1 resource not found, build、create and return
// 2.2 resource found, update controlee and update status
type SetupTemplate struct {
	context.Context
	BasicReconciler

	// callbacks callback functions
	callbacks *SetupCallbacks
}

// SetupContext holds callback parameters
type SetupContext struct {
	logr.Logger
	context.Context
}

func (c SetupContext) WithValues(keysAndValues ...interface{}) SetupContext {
	return SetupContext{Logger: c.Logger.WithValues(keysAndValues), Context: c.Context}
}

func (c SetupContext) FormatInfo(message string, params ...interface{}) {
	c.Info(fmt.Sprintf(message, params...))
}

func (c SetupContext) FormatError(err error, message string, params ...interface{}) {
	c.Error(err, fmt.Sprintf(message, params...))
}

// SetupResource make the resource meets its ready state
// return (ready, requeue)
func (t SetupTemplate) SetupResource(owner client.Object, resource client.Object, name string, params ...interface{}) (bool, time.Duration) {

	// do nothing if resource is being deleted
	if owner.GetDeletionTimestamp() != nil {
		return false, RetryNever
	}

	namespace := owner.GetNamespace()
	meta := ParseResourceMeta(resource, namespace, name)
	logger := t.ResourceLogger(resource, namespace, name)
	ctx := SetupContext{Logger: logger, Context: t.Context}

	// check if the resource already exists, if not create a new one
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := t.Client.Get(t, namespacedName, resource); err != nil {
		if errors.IsNotFound(err) {
			// should create resource if not found
			logger.Info("+++ create missing resource")
			if t.callbacks.BuildResource == nil {
				return false, RetryNever // ignore everything
			} else if resource, err = t.callbacks.BuildResource(ctx, owner, name, params...); err != nil {
				message := fmt.Sprintf("Build resource %s failed", meta.GeneralName)
				t.record(ctx, owner, ConditionStatusBuildFailed, message, err)
			} else if _, err = t.setOwnerReference(owner, resource); err != nil {
				message := fmt.Sprintf("Set owner reference to %s failed", meta.GeneralName)
				t.record(ctx, owner, ConditionStatusReferFailed, message, err)
			} else if err = t.Client.Create(t, resource); err == nil {
				message := fmt.Sprintf("Resource %s created", meta.GeneralName)
				t.record(ctx, owner, ConditionStatusCreateSuccess, message, nil)
				return false, RetryNever // reconciled by watching
			} else if errors.IsAlreadyExists(err) { // ignore this
				message := fmt.Sprintf("Resource %s already exists", meta.GeneralName)
				t.record(ctx, owner, ConditionStatusAlreadyExists, message, nil)
			} else {
				message := fmt.Sprintf("Resource %s create failed", meta.GeneralName)
				t.record(ctx, owner, ConditionStatusCreateFailed, message, err)
			}
		} else {
			message := fmt.Sprintf("Resource %s fetch failed", meta.GeneralName)
			t.record(ctx, owner, ConditionStatusFetchFailed, message, err)
		}
		return false, RetryLater // always reconcile, with errors
	}

	logger.Info(fmt.Sprintf("%s already exists", meta.Kind))

	// update reference if the owner of the resource is missing
	ownerChanged, err := t.setOwnerReference(owner, resource)
	if err != nil {
		logger.Info("set owner reference of resource failed")
		message := fmt.Sprintf("Set owner reference to %s failed", meta.GeneralName)
		t.record(ctx, owner, ConditionStatusReferFailed, message, err)
		return false, RetryLater
	} else if ownerChanged {
		logger.Info("*** owner reference of resource updated")
	}

	// check changes between owner spec and resource
	resourceUpdated := false
	if t.callbacks.UpdateResource != nil {
		if resourceUpdated, err = t.callbacks.UpdateResource(ctx, owner, resource, params...); err != nil {
			logger.Info("update spec of resource failed")
			message := fmt.Sprintf("Update %s resource spec failed", meta.GeneralName)
			t.record(ctx, owner, ConditionStatusUpdateFailed, message, err)
			return false, RetryLater
		} else if resourceUpdated {
			logger.Info("*** spec of resource updated")
		}
	}

	if ownerChanged || resourceUpdated {
		if err := t.Client.Update(t, resource); err != nil {
			message := fmt.Sprintf("Resource %s update failed", meta.GeneralName)
			t.record(ctx, owner, ConditionStatusUpdateFailed, message, err)
			return false, RetryLater
		} else {
			message := fmt.Sprintf("Resource %s updated", meta.GeneralName)
			t.record(ctx, owner, ConditionStatusUpdateSuccess, message, nil)
			return false, RetryNever // reconciled by watching
		}
	}

	// update owner status
	if t.callbacks.UpdateOwnerStatus != nil {
		meta := ParseResourceMeta(owner)
		logger := t.ResourceLogger(owner)
		if updated, err := t.callbacks.UpdateOwnerStatus(ctx, owner, resource, params...); err != nil {
			logger.Info("update resource status failed")
			message := fmt.Sprintf("Resource %s update failed", meta.GeneralName)
			t.record(ctx, owner, ConditionStatusUpdateFailed, message, err)
			return false, RetryLater
		} else if updated {
			logger.Info("*** resource status has changed")
			if err := t.Client.Status().Update(t, owner); err != nil {
				return false, RetryLater // explicit reconciled
			}
			return false, RetryNever // resource updated and implicit reconciled
		}
	}

	// everything is ok, ready=true can only return here
	return true, RetryNever // resource is ready and no need to reconcile
}

func (t SetupTemplate) record(ctx SetupContext, client client.Object, status string, message string, err error) {
	if err == nil {
		ctx.Info(message)
		t.Recorder.Event(client, v1.EventTypeNormal, status, message)
	} else {
		ctx.Error(err, message)
		info := fmt.Sprintf("%s: %s", message, err.Error())
		t.Recorder.Event(client, v1.EventTypeWarning, status, info)
	}
}

func (t SetupTemplate) setOwnerReference(owner metav1.Object, resource metav1.Object) (bool, error) {
	if !t.callbacks.NotOwnerReferred && owner.GetDeletionTimestamp() == nil { // must check this flag
		if controller := metav1.GetControllerOf(resource); controller == nil {
			return true, ctrl.SetControllerReference(owner, resource, t.Scheme)
		}
	}
	return false, nil
}

type ResourceMeta struct {
	Kind           string
	GeneralName    string
	NamespacedName types.NamespacedName
}

func ParseResourceMeta(resource client.Object, args ...interface{}) ResourceMeta {
	namespacedName := ParseNamespacedName(resource, args...)
	resource.SetName(namespacedName.Name)
	resource.SetNamespace(namespacedName.Namespace)
	kind := reflect.TypeOf(resource).Elem().Name()
	return ResourceMeta{
		Kind:           kind,
		GeneralName:    fmt.Sprintf("%s(%s)", kind, namespacedName),
		NamespacedName: namespacedName,
	}
}

// ParseNamespacedName return namespacedName
func ParseNamespacedName(resource client.Object, args ...interface{}) types.NamespacedName {
	var name types.NamespacedName
	if args == nil || len(args) == 0 {
		name.Name = resource.GetName()
		name.Namespace = resource.GetNamespace()
	} else if len(args) == 1 {
		name = args[0].(types.NamespacedName)
	} else if len(args) >= 2 {
		name.Namespace = args[0].(string)
		name.Name = args[1].(string)
	}
	return name
}
