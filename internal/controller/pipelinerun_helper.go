package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/konflux-ci/operator-toolkit/metadata"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const NotificationPipelineRunFinalizer string = "konflux.ci/notification"
const NotificationPipelineRunAnnotation string = "konflux.ci/notified"
const NotificationPipelineRunAnnotationValue string = "true"
const PipelineRunTypeLabel string = "pipelinesascode.tekton.dev/event-type"
const PushPipelineRunTypeValue string = "push"
const AppLabelKey string = "appstudio.openshift.io/application"

// AddFinalizerToPipelineRun adds the finalizer to the PipelineRun.
// If finalizer was not added successfully, a non-nil error is returned.
func AddFinalizerToPipelineRun(ctx context.Context, pipelineRun *tektonv1.PipelineRun, r *NotificationServiceReconciler, finalizer string) error {
	r.Log.Info("Adding finalizer", "pipelineRun", pipelineRun.Name)
	patch := client.MergeFrom(pipelineRun.DeepCopy())
	if ok := controllerutil.AddFinalizer(pipelineRun, finalizer); ok {
		err := r.Client.Patch(ctx, pipelineRun, patch)
		if err != nil {
			return fmt.Errorf("error occurred while patching the updated PipelineRun after finalizer addition: %w", err)
		}
		r.Log.Info("Finalizer was added", "pipelineRun", pipelineRun.Name)
	}
	return nil
}

// RemoveFinalizerFromPipelineRun removes the finalizer from the PipelineRun.
// If finalizer was not removed successfully, a non-nil error is returned.
func RemoveFinalizerFromPipelineRun(ctx context.Context, pipelineRun *tektonv1.PipelineRun, r *NotificationServiceReconciler, finalizer string) error {
	r.Log.Info("Removing finalizer", "pipelineRun", pipelineRun.Name)
	patch := client.MergeFrom(pipelineRun.DeepCopy())
	if ok := controllerutil.RemoveFinalizer(pipelineRun, finalizer); ok {
		err := r.Client.Patch(ctx, pipelineRun, patch)
		if err != nil {
			return fmt.Errorf("error occurred while patching the updated PipelineRun after finalizer removal: %w", err)
		}
		r.Log.Info("Finalizer was removed", "pipelineRun", pipelineRun.Name)
	}
	return nil
}

// GetResultsFromPipelineRun extracts results from pipelinerun
// And adds the pipelinerunName
// Return error if failed to extract results or if results does not exist
func GetResultsFromPipelineRun(pipelineRun *tektonv1.PipelineRun) ([]byte, error) {
	namedResults := []tektonv1.PipelineRunResult{
		{
			Name:  "PIPELINERUN_NAME",
			Value: *tektonv1.NewStructuredValues(pipelineRun.Name),
		},
		{
			Name:  "NAMESPACE",
			Value: *tektonv1.NewStructuredValues(pipelineRun.Namespace),
		},
		{
			Name:  "APPLICATION",
			Value: *tektonv1.NewStructuredValues(GetApplicationNameFromPipelineRun(pipelineRun)),
		},
	}
	fetchedResults := pipelineRun.Status.Results
	fullResults := append(namedResults, fetchedResults...)
	results, err := json.Marshal(fullResults)
	if err != nil {
		return nil, fmt.Errorf("failed to marshel results from pipelinerun %s: %w", pipelineRun.Name, err)
	}
	return results, nil
}

// AddNotificationAnnotationToPipelineRun adds an annotation to the PipelineRun.
// If annotation was not added successfully, a non-nil error is returned.
func AddAnnotationToPipelineRun(ctx context.Context, pipelineRun *tektonv1.PipelineRun, r *NotificationServiceReconciler, annotation string, annotationValue string) error {
	r.Log.Info("Adding annotation", "pipelineRun", pipelineRun.Name)
	patch := client.MergeFrom(pipelineRun.DeepCopy())
	err := metadata.SetAnnotation(&pipelineRun.ObjectMeta, annotation, annotationValue)
	if err != nil {
		return fmt.Errorf("error occurred while setting the annotation: %w", err)
	}
	err = r.Client.Patch(ctx, pipelineRun, patch)
	if err != nil {
		r.Log.Error(err, "Error in update annotation patching", "pipelineRun", pipelineRun.Name)
		return fmt.Errorf("error occurred while patching the updated pipelineRun after annotation addition: %w", err)
	}
	r.Log.Info("Annotation was added", "pipelineRun", pipelineRun.Name)
	return nil
}

// IsFinalizerExistInPipelineRun checks if an finalizer exists in pipelineRun
// Return true if yes, otherwise return false
// If the object passed to this function is not a PipelineRun, the function will return false.
func IsFinalizerExistInPipelineRun(object client.Object, finalizer string) bool {
	if pipelineRun, ok := object.(*tektonv1.PipelineRun); ok {
		return controllerutil.ContainsFinalizer(pipelineRun, finalizer)
	}
	return false
}

// IsPipelineRunEndedSuccessfully returns a boolean indicating whether the PipelineRun succeeded or not.
// If the object passed to this function is not a PipelineRun, the function will return false.
func IsPipelineRunEndedSuccessfully(object client.Object) bool {
	if pipelineRun, ok := object.(*tektonv1.PipelineRun); ok {
		return pipelineRun.Status.GetCondition(apis.ConditionSucceeded).IsTrue()
	}
	return false
}

// IsPipelineRunEnded returns a boolean indicating whether the PipelineRun finished or not.
// If the object passed to this function is not a PipelineRun, the function will return false.
func IsPipelineRunEnded(object client.Object) bool {
	if pr, ok := object.(*tektonv1.PipelineRun); ok {
		return !pr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown()
	}

	return false
}

// IsNotificationAnnotationExist checks if an annotation exists in pipelineRun
// Return true if yes, otherwise return false
// If the object passed to this function is not a PipelineRun, the function will return false.
func IsAnnotationExistInPipelineRun(object client.Object, annotation string, annotationValue string) bool {
	if pipelineRun, ok := object.(*tektonv1.PipelineRun); ok {
		return metadata.HasAnnotationWithValue(pipelineRun, annotation, annotationValue)
	}
	return false
}

// ShouldProcessPipelineRun checks whether a PipelineRun matches the filters configured via
// NOTIFICATION_FILTER_LABELS and NOTIFICATION_FILTER_ANNOTATIONS. Defaults to push-only when both are unset.
func ShouldProcessPipelineRun(object client.Object) bool {
	pipelineRun, ok := object.(*tektonv1.PipelineRun)
	if !ok {
		return false
	}

	filterLabels := os.Getenv("NOTIFICATION_FILTER_LABELS")
	filterAnnotations := os.Getenv("NOTIFICATION_FILTER_ANNOTATIONS")

	if filterLabels == "" && filterAnnotations == "" {
		return metadata.HasLabelWithValue(pipelineRun, PipelineRunTypeLabel, PushPipelineRunTypeValue)
	}

	if filterLabels != "" {
		labelPairs := strings.Split(filterLabels, ",")
		for _, pair := range labelPairs {
			pair = strings.TrimSpace(pair)
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				ctrl.Log.Info("Ignoring malformed label filter entry, expected key=value format", "entry", pair)
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if metadata.HasLabelWithValue(pipelineRun, key, value) {
				return true
			}
		}
	}

	if filterAnnotations != "" {
		for _, key := range strings.Split(filterAnnotations, ",") {
			key = strings.TrimSpace(key)
			if metadata.HasAnnotation(pipelineRun, key) {
				return true
			}
		}
	}

	return false
}

// GetApplicationNameFromPipelineRun gets the application name from the application label
// Returns the application name or empty string in case the applicatio name could not retrieved
func GetApplicationNameFromPipelineRun(pipelineRun *tektonv1.PipelineRun) string {
	appLabel, err := metadata.GetLabelsWithPrefix(pipelineRun, AppLabelKey)
	if err != nil {
		return ""
	}
	return appLabel[AppLabelKey]
}
