/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/konflux-ci/notification-service/pkg/notifier"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NotificationServiceReconciler reconciles a NotificationService object
type NotificationServiceReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Notifier notifier.Notifier
}

// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns/finalizers,verbs=update

// Reconcile will monitor the pipelinerun, extract its result and send it as a webhook
// When a pipelinerun is created, it will add a finalizer to it so we will be able to extract the results
// After a pipelinerun ends successfully, the results will be extracted from it and will be sent as a webhook,
// An annotation will be added to mark this pipelinerun as handled and the finalizer will be rmoved
// to allow the deletion of this pipelinerun
func (r *NotificationServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithName("Notification controller")
	pipelineRun := &tektonv1.PipelineRun{}

	err := r.Get(ctx, req.NamespacedName, pipelineRun)
	if err != nil {
		logger.Error(err, "Failed to get pipelineRun for", "req", req.NamespacedName)
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling PipelineRun", "Name", pipelineRun.Name)
	if !IsPipelineRunEnded(pipelineRun) &&
		!IsFinalizerExistInPipelineRun(pipelineRun, NotificationPipelineRunFinalizer) {
		err = AddFinalizerToPipelineRun(ctx, pipelineRun, r, NotificationPipelineRunFinalizer)
		if err != nil {
			logger.Error(err, "Failed to add finalizer", "pipelineRun", pipelineRun.Name)
			return ctrl.Result{}, err
		}
	}

	if IsPipelineRunEnded(pipelineRun) {
		logger.Info("Pipelinerun ended", "pipelinerun", pipelineRun.Name, "ended_successfully", IsPipelineRunEndedSuccessfully(pipelineRun))
		if IsPipelineRunEndedSuccessfully(pipelineRun) && !IsAnnotationExistInPipelineRun(pipelineRun, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue) {
			results, err := GetResultsFromPipelineRun(pipelineRun)
			if err != nil {
				logger.Error(err, "Failed to get results", "pipelineRun", pipelineRun.Name)
				return ctrl.Result{}, err
			}
			err = r.Notifier.Notify(ctx, string(results))
			notifications.Inc()
			if err != nil {
				logger.Error(err, "Failed to Notify")
				notificationsFailures.Inc()
				return ctrl.Result{}, err
			}
			logger.Info("SNS Notified", "message", string(results))
			err = AddAnnotationToPipelineRun(ctx, pipelineRun, r, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
			if err != nil {
				logger.Error(err, "Failed to add annotation", "pipelineRun", pipelineRun.Name)
				return ctrl.Result{}, err
			}
		}
		err = RemoveFinalizerFromPipelineRun(ctx, pipelineRun, r, NotificationPipelineRunFinalizer)
		if err != nil {
			logger.Error(err, "Failed to remove finalizer", "pipelineRun", pipelineRun.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NotificationServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tektonv1.PipelineRun{}).
		WithEventFilter(predicate.Or(
			PushPipelineRunCreatedPredicate(),
			PushPipelineRunEndedFinalizerPredicate(),
			PushPipelineRunEndedNoAnnotationPredicate(),
		)).
		Complete(r)
}
