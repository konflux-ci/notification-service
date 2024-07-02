package controller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/konflux-ci/operator-toolkit/metadata"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const NotificationPipelineRunFinalizer string = "konflux.ci/notification"
const NotificationPipelineRunAnnotation string = "konflux.ci/notified"
const NotificationPipelineRunAnnotationValue string = "true"

// AddFinalizerToPipelineRun adds the finalizer to the PipelineRun.
// If finalizer was not added successfully, a non-nil error is returned.
func AddFinalizerToPipelineRun(ctx context.Context, pipelineRun *tektonv1.PipelineRun, r *NotificationServiceReconciler, finalizer string) error {
	r.Log.Info("Adding finalizer")
	patch := client.MergeFrom(pipelineRun.DeepCopy())
	if ok := controllerutil.AddFinalizer(pipelineRun, finalizer); ok {
		err := r.Client.Patch(ctx, pipelineRun, patch)
		if err != nil {
			return fmt.Errorf("Error occurred while patching the updated PipelineRun after finalizer addition: %w", err)
		}
		r.Log.Info("Finalizer was added to PipelineRun %s", pipelineRun.Name)
	}
	return nil
}

// RemoveFinalizerFromPipelineRun removes the finalizer from the PipelineRun.
// If finalizer was not removed successfully, a non-nil error is returned.
func RemoveFinalizerFromPipelineRun(ctx context.Context, pipelineRun *tektonv1.PipelineRun, r *NotificationServiceReconciler, finalizer string) error {
	r.Log.Info("Removing finalizer")
	patch := client.MergeFrom(pipelineRun.DeepCopy())
	if ok := controllerutil.RemoveFinalizer(pipelineRun, finalizer); ok {
		err := r.Client.Patch(ctx, pipelineRun, patch)
		if err != nil {
			return fmt.Errorf("Error occurred while patching the updated PipelineRun after finalizer removal: %w", err)
		}
		r.Log.Info("Finalizer was removed from %s", pipelineRun.Name)
	}
	return nil
}

// GetResultsFromPipelineRun extracts results from pipelinerun
// Return error if failed to extract results
func GetResultsFromPipelineRun(pipelineRun *tektonv1.PipelineRun) ([]byte, error) {
	results, err := json.Marshal(pipelineRun.Status.Results)
	if err != nil {
		return nil, fmt.Errorf("Failed to get results from pipelinerun %s: %w", pipelineRun.Name, err)
	}
	return results, nil
}

// AddNotificationAnnotationToPipelineRun adds an annotation to the PipelineRun.
// If annotation was not added successfully, a non-nil error is returned.
func AddAnnotationToPipelineRun(ctx context.Context, pipelineRun *tektonv1.PipelineRun, r *NotificationServiceReconciler, annotation string, annotationValue string) error {
	r.Log.Info("Adding annotation")
	patch := client.MergeFrom(pipelineRun.DeepCopy())
	err := metadata.SetAnnotation(&pipelineRun.ObjectMeta, annotation, annotationValue)
	if err != nil {
		return fmt.Errorf("Error occurred while setting the annotation: %w", err)
	}
	err = r.Client.Patch(ctx, pipelineRun, patch)
	if err != nil {
		r.Log.Info("Error in update annotation client: %s", err)
		return fmt.Errorf("Error occurred while patching the updated pipelineRun after annotation addition: %w", err)
	}
	r.Log.Info("Annotation was added to %s", pipelineRun.Name)
	return nil
}

// IsFinalizerExistInPipelineRun checks if an finalizer exists in pipelineRun
// Return true if yes, otherwise return false
func IsFinalizerExistInPipelineRun(pipelineRun *tektonv1.PipelineRun, finalizer string) bool {
	return controllerutil.ContainsFinalizer(pipelineRun, finalizer)
}

// IsPipelineRunEndedSuccessfully returns a boolean indicating whether the PipelineRun succeeded or not.
func IsPipelineRunEndedSuccessfully(pipelineRun *tektonv1.PipelineRun) bool {
	return pipelineRun.Status.GetCondition(apis.ConditionSucceeded).IsTrue()
}

// IsNotificationAnnotationExist checks if an annotation exists in pipelineRun
// Return true if yes, otherwise return false
func IsAnnotationExistInPipelineRun(pipelineRun *tektonv1.PipelineRun, annotation string, annotationValue string) bool {
	return metadata.HasAnnotationWithValue(pipelineRun, annotation, annotationValue)
}
