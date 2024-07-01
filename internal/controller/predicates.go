package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PushPipelineRunCreatedPredicate returns a predicate which filters out all objects except
// Push PipelineRuns that have just created
func PushPipelineRunCreatedPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return IsPushPipelineRun(createEvent.Object)
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

// PushPipelineRunEndedNoAnnotationPredicate returns a predicate which filters out all objects except
// Push PipelineRuns which have finished and has no notification annotation
func PushPipelineRunEndedNoAnnotationPredicate() predicate.Predicate {
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
			return (IsPushPipelineRun(e.ObjectNew) &&
				IsPipelineRunEnded(e.ObjectNew) &&
				!IsAnnotationExistInPipelineRun(e.ObjectNew, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue))
		},
	}
}

// PushPipelineRunEndedFinalizerPredicate returns a predicate which filters out all objects except
// Push PipelineRuns which have finished and has finalizer
func PushPipelineRunEndedFinalizerPredicate() predicate.Predicate {
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
				IsPushPipelineRun(e.ObjectNew) &&
				IsPipelineRunEnded(e.ObjectNew))
		},
	}
}
