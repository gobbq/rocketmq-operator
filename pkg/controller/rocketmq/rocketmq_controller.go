/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package rocketmq contains the implementation of the RocketMQ CRD reconcile function
package rocketmq

import (
	"context"
	rocketmqv1alpha1 "github.com/apache/rocketmq-operator/pkg/apis/rocketmq/v1alpha1"
	cons "github.com/apache/rocketmq-operator/pkg/constants"
	"github.com/apache/rocketmq-operator/pkg/tool"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("controller_rocketmq")
var aWhile = 5 * time.Second

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// SetupWithManager creates a new RocketMQ Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func SetupWithManager(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRocketMQ{
		BasicReconciler: tool.NewBasicReconciler(mgr, "controller_rocketmq"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rocketmq-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RocketMQ
	err = c.Watch(&source.Kind{Type: &rocketmqv1alpha1.RocketMQ{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rocketmqv1alpha1.NameService{}}, &handler.EnqueueRequestForOwner{OwnerType: &rocketmqv1alpha1.RocketMQ{}})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rocketmqv1alpha1.Broker{}}, &handler.EnqueueRequestForOwner{OwnerType: &rocketmqv1alpha1.RocketMQ{}})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rocketmqv1alpha1.Console{}}, &handler.EnqueueRequestForOwner{OwnerType: &rocketmqv1alpha1.RocketMQ{}})
	if err != nil {
		return err
	}

	return nil
}

//+kubebuilder:rbac:groups=rocketmq.apache.org,resources=rocketmqs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rocketmq.apache.org,resources=rocketmqs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rocketmq.apache.org,resources=rocketmqs/finalizers,verbs=update
//+kubebuilder:rbac:groups=rocketmq.apache.org,resources=nameservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rocketmq.apache.org,resources=brokers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rocketmq.apache.org,resources=consoles,verbs=get;list;watch;create;update;patch;delete

// ReconcileRocketMQ reconciles a RocketMQ object
type ReconcileRocketMQ struct {
	tool.BasicReconciler
}

// Reconcile reads that state of the cluster for a RocketMQ object and makes changes based on the state read
// and what is in the RocketMQ.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRocketMQ) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling RocketMQ")

	// Fetch the RocketMQ rmq
	rmq := rocketmqv1alpha1.RocketMQ{}
	if missing, err := r.FetchResource(ctx, &rmq, req.NamespacedName); missing {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{RequeueAfter: tool.RetryLater}, err
	}

	// setup name service resource
	if ready, duration := r.setupNameServiceResource(ctx, &rmq); !ready {
		return ctrl.Result{RequeueAfter: duration}, nil
	}

	// setup broker clusters resource if name service ready
	if ready, duration := r.setupBrokerClustersResources(ctx, &rmq); !ready {
		return ctrl.Result{RequeueAfter: duration}, nil
	}

	// setup console resource if name service ready
	if ready, duration := r.setupConsoleResource(ctx, &rmq); !ready {
		return ctrl.Result{RequeueAfter: duration}, nil
	}

	return reconcile.Result{}, nil
}

func (r ReconcileRocketMQ) setupNameServiceResource(ctx context.Context, rmq *rocketmqv1alpha1.RocketMQ) (bool, time.Duration) {
	name := rmq.Spec.NameService.Name
	if len(name) == 0 {
		name = tool.NameServiceName(rmq.Name)
	}

	return r.NewSetupTemplate(ctx, tool.SetupCallbacks{
		NotOwnerReferred:  false,
		BuildResource:     r.buildNameServiceResource,
		UpdateResource:    r.updateNameServiceResource,
		UpdateOwnerStatus: r.updateNameServiceOwnerStatus,
	}).SetupResource(rmq, &rocketmqv1alpha1.NameService{}, name)
}

func (r *ReconcileRocketMQ) buildNameServiceResource(ctx tool.SetupContext, owner client.Object, name string, params ...interface{}) (client.Object, error) {
	rmq := owner.(*rocketmqv1alpha1.RocketMQ)
	nameService := rocketmqv1alpha1.NameService{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  rmq.Namespace,
			Finalizers: []string{metav1.FinalizerOrphanDependents},
		},
		Spec: rmq.Spec.NameService.Spec,
	}
	return &nameService, nil
}

func (r *ReconcileRocketMQ) updateNameServiceResource(ctx tool.SetupContext, owner client.Object, resource client.Object, params ...interface{}) (bool, error) {
	nameService := resource.(*rocketmqv1alpha1.NameService)
	if nameService.ObjectMeta.Annotations == nil {
		nameService.ObjectMeta.Annotations = make(map[string]string)
	}
	if _, ok := nameService.ObjectMeta.Annotations[cons.RocketMQNameAnnotation]; !ok {
		rmq := owner.(*rocketmqv1alpha1.RocketMQ)
		ctx.Info("Update NameService resource annotation.", cons.RocketMQNameAnnotation, rmq.Name)
		nameService.ObjectMeta.Annotations[cons.RocketMQNameAnnotation] = rmq.Name
		return true, nil
	}
	return false, nil
}

func (r *ReconcileRocketMQ) updateNameServiceOwnerStatus(sc tool.SetupContext, owner client.Object, resource client.Object, params ...interface{}) (bool, error) {
	rmq := owner.(*rocketmqv1alpha1.RocketMQ)
	nameService := resource.(*rocketmqv1alpha1.NameService)

	servers := nameService.Status.NameServers
	if !tool.IsSameList(rmq.Status.NameServers, servers) {
		rmq.Status.NameServers = servers
		rmq.Status.NameServiceReady = int32(len(servers))
		return true, nil
	}
	return false, nil
}

func (r ReconcileRocketMQ) setupBrokerClustersResources(ctx context.Context, rmq *rocketmqv1alpha1.RocketMQ) (bool, time.Duration) {

	if len(rmq.Status.NameServers) != int(rmq.Spec.NameService.Spec.Size) {
		return true, tool.RetryNever
	}

	// checkout broker cluster list
	var brokers []rocketmqv1alpha1.BrokerClusterConfig
	replica := rmq.Spec.BrokerClusters.Replica
	template := rmq.Spec.BrokerClusters.Template
	if replica != nil && template != nil {
		for i := 0; i < *replica; i++ {
			name := tool.BrokerName(rmq.Name, i)
			brokers = append(brokers, rocketmqv1alpha1.BrokerClusterConfig{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
				Spec:                 *template,
			})
		}
	}
	if rmq.Spec.BrokerClusters.Items != nil {
		brokers = append(brokers, rmq.Spec.BrokerClusters.Items...)
	}

	// check each Broker resource is created
	readyBrokers := 0
	retryOrNot := tool.RetryNever
	for _, item := range brokers {
		createTpl := r.NewSetupTemplate(ctx, tool.SetupCallbacks{
			BuildResource:  r.buildBrokerResource,
			UpdateResource: r.updateBrokerResource,
		})
		ready, requeue := createTpl.SetupResource(rmq, &rocketmqv1alpha1.Broker{}, item.Name, &item.Spec)

		// count brokers in ready status
		if ready {
			readyBrokers++ // ignore sts status
		}
		if requeue == tool.RetryLater {
			retryOrNot = tool.RetryLater
		}
	}

	// update rocketmq status
	rmq.Status.BrokerClusterReady = int32(readyBrokers)
	if err := r.Client.Status().Update(ctx, rmq); err != nil {
		return false, tool.RetryLater
	}
	return readyBrokers == len(brokers), retryOrNot
}

func (r ReconcileRocketMQ) buildBrokerResource(ctx tool.SetupContext, owner client.Object, name string, params ...interface{}) (client.Object, error) {
	rmq := owner.(*rocketmqv1alpha1.RocketMQ)
	spec := params[0].(*rocketmqv1alpha1.BrokerSpec)

	broker := rocketmqv1alpha1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  rmq.Namespace,
			Finalizers: []string{metav1.FinalizerOrphanDependents},
		},
		Spec: *spec,
	}
	return &broker, nil
}

func (r *ReconcileRocketMQ) updateBrokerResource(sc tool.SetupContext, owner client.Object, resource client.Object, params ...interface{}) (bool, error) {
	broker := resource.(*rocketmqv1alpha1.Broker)
	if broker.ObjectMeta.Annotations == nil {
		broker.ObjectMeta.Annotations = make(map[string]string)
	}
	if _, ok := broker.ObjectMeta.Annotations[cons.RocketMQNameAnnotation]; !ok {
		rmq := owner.(*rocketmqv1alpha1.RocketMQ)
		broker.ObjectMeta.Annotations[cons.RocketMQNameAnnotation] = rmq.Name
		return true, nil
	}
	return false, nil
}

func (r ReconcileRocketMQ) setupConsoleResource(ctx context.Context, rmq *rocketmqv1alpha1.RocketMQ) (bool, time.Duration) {

	if len(rmq.Status.NameServers) != int(rmq.Spec.NameService.Spec.Size) {
		return true, tool.RetryNever
	}

	name := rmq.Spec.Console.Name
	if len(name) == 0 {
		name = tool.ConsoleName(rmq.Name)
	}

	return r.NewSetupTemplate(ctx, tool.SetupCallbacks{
		BuildResource:     r.buildConsoleResource,
		UpdateResource:    r.updateConsoleResource,
		UpdateOwnerStatus: r.updateConsoleOwnerStatus,
	}).SetupResource(rmq, &rocketmqv1alpha1.Console{}, name)
}

func (r *ReconcileRocketMQ) buildConsoleResource(ctx tool.SetupContext, owner client.Object, name string, params ...interface{}) (client.Object, error) {
	rmq := owner.(*rocketmqv1alpha1.RocketMQ)
	return &rocketmqv1alpha1.Console{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  rmq.Namespace,
			Finalizers: []string{metav1.FinalizerOrphanDependents},
		},
		Spec: rmq.Spec.Console.Spec,
	}, nil
}

func (r *ReconcileRocketMQ) updateConsoleResource(sc tool.SetupContext, owner client.Object, resource client.Object, params ...interface{}) (bool, error) {
	console := resource.(*rocketmqv1alpha1.Console)
	if console.ObjectMeta.Annotations == nil {
		console.ObjectMeta.Annotations = make(map[string]string)
	}
	if _, ok := console.ObjectMeta.Annotations[cons.RocketMQNameAnnotation]; !ok {
		rmq := owner.(*rocketmqv1alpha1.RocketMQ)
		console.ObjectMeta.Annotations[cons.RocketMQNameAnnotation] = rmq.Name
		return true, nil
	}
	return false, nil
}

func (r *ReconcileRocketMQ) updateConsoleOwnerStatus(ctx tool.SetupContext, owner client.Object, resource client.Object, params ...interface{}) (bool, error) {
	rmq := owner.(*rocketmqv1alpha1.RocketMQ)
	console := resource.(*rocketmqv1alpha1.Console)

	readyReplicas := console.Status.ReadyReplicas
	if rmq.Status.ConsoleReady != readyReplicas {
		rmq.Status.ConsoleReady = readyReplicas
		return true, nil
	}
	return false, nil
}
