package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PipelineRunCreatedPredicate returns a predicate which filters out all objects except
// PipelineRuns matching the configured filters that have just been created
func PipelineRunCreatedPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return ShouldProcessPipelineRun(createEvent.Object)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
	}
}

// PipelineRunEndedNoAnnotationPredicate returns a predicate which filters out all objects except
// matching PipelineRuns which have finished and have no notification annotation
func PipelineRunEndedNoAnnotationPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return (ShouldProcessPipelineRun(e.ObjectNew) &&
				IsPipelineRunEnded(e.ObjectNew) &&
				!IsAnnotationExistInPipelineRun(e.ObjectNew, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue))
		},
	}
}

// PipelineRunEndedFinalizerPredicate returns a predicate which filters out all objects except
// matching PipelineRuns which have finished and have a finalizer
func PipelineRunEndedFinalizerPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return (IsFinalizerExistInPipelineRun(e.ObjectNew, NotificationPipelineRunFinalizer) &&
				ShouldProcessPipelineRun(e.ObjectNew) &&
				IsPipelineRunEnded(e.ObjectNew))
		},
	}
}

// PipelineRunDeletingPredicate returns a predicate which filters out all objects except
// matching PipelineRuns which have been updated to deleting
func PipelineRunDeletingPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if ShouldProcessPipelineRun(e.ObjectNew) &&
				e.ObjectOld.GetDeletionTimestamp() == nil &&
				e.ObjectNew.GetDeletionTimestamp() != nil {
				return true
			}
			return false
		},
	}
}
