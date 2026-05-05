package controller

import (
	"encoding/json"
	"time"

	"github.com/konflux-ci/operator-toolkit/metadata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	v1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Unit testing for pipelinerun_helper", func() {
	var (
		testTruePipelineRun,
		testFalsePipelineRun,
		testErrMmarshelPipelineRun,
		testResourcesPipelineRun *tektonv1.PipelineRun
		notPipelineRun *tektonv1.TaskRun
	)
	const (
		timeout                      = time.Second * 10
		interval                     = time.Millisecond * 250
		pushPipelineRunName          = "push-pipelinerun-sample-unit-test"
		testResourcesPipelineRunName = "test-resources-pipelinerun-unit-test"
		namespace                    = "default"
	)

	testPipelineLookupKey := types.NamespacedName{Name: testResourcesPipelineRunName, Namespace: namespace}
	testPipelineRun := &tektonv1.PipelineRun{}

	Describe("Push Pipelinerun; Ended; with finalizer, annotation, results and application name", func() {
		testTruePipelineRun = &tektonv1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pushPipelineRunName,
				Namespace: namespace,
				Labels: map[string]string{
					PipelineRunTypeLabel: PushPipelineRunTypeValue,
					AppLabelKey:          "myapp",
				},
				Finalizers: []string{
					NotificationPipelineRunFinalizer,
				},
				Annotations: map[string]string{
					NotificationPipelineRunAnnotation: NotificationPipelineRunAnnotationValue,
				},
			},
			Spec: tektonv1.PipelineRunSpec{
				PipelineRef: &tektonv1.PipelineRef{},
			},
			Status: tektonv1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: time.Now()},
					CompletionTime: &metav1.Time{Time: time.Now().Add(5 * time.Minute)},
					Results: []tektonv1.PipelineRunResult{
						{
							Name:  "IMAGE_DIGEST",
							Value: *tektonv1.NewStructuredValues("image_digest_value"),
						},
						{
							Name:  "IMAGE_URL",
							Value: *tektonv1.NewStructuredValues("image"),
						},
						{
							Name:  "CHAINS-GIT_URL",
							Value: *tektonv1.NewStructuredValues("git_url_value"),
						},
						{
							Name:  "CHAINS-GIT_COMMIT",
							Value: *tektonv1.NewStructuredValues("git_commit_value"),
						},
					},
				},
				Status: v1.Status{
					Conditions: v1.Conditions{
						apis.Condition{
							Message: "Tasks Completed: 12 (Failed: 0, Cancelled 0), Skipped: 2",
							Reason:  "Completed",
							Status:  "True",
							Type:    apis.ConditionSucceeded,
						},
					},
				},
			},
		}
		Context("when a push pipelinerun ends successfully and has results", func() {
			It("ShouldProcessPipelineRun should return true", func() {
				got := ShouldProcessPipelineRun(testTruePipelineRun)
				Expect(got).To(BeTrue())
			})
			It("IsAnnotationExistInPipelineRun should return true", func() {
				got := IsAnnotationExistInPipelineRun(testTruePipelineRun, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
				Expect(got).To(BeTrue())
			})
			It("IsFinalizerExistInPipelineRun should return true", func() {
				got := IsFinalizerExistInPipelineRun(testTruePipelineRun, NotificationPipelineRunFinalizer)
				Expect(got).To(BeTrue())
			})
			It("IsPipelineRunEnded should return true", func() {
				got := IsPipelineRunEnded(testTruePipelineRun)
				Expect(got).To(BeTrue())
			})
			It("IsPipelineRunEndedSuccessfully should return true", func() {
				got := IsPipelineRunEndedSuccessfully(testTruePipelineRun)
				Expect(got).To(BeTrue())
			})
			It("GetApplicationNameFromPipelineRun should extract the application name", func() {
				got := GetApplicationNameFromPipelineRun(testTruePipelineRun)
				Expect(got).To(Equal("myapp"))
			})
			It("GetResultsFromPipelineRun should extract the result and add it the pipelinerunName", func() {
				got, err := GetResultsFromPipelineRun(testTruePipelineRun)
				Expect(err).ToNot(HaveOccurred())
				result := []tektonv1.PipelineRunResult{
					{
						Name:  "PIPELINERUN_NAME",
						Value: *tektonv1.NewStructuredValues(pushPipelineRunName),
					},
					{
						Name:  "NAMESPACE",
						Value: *tektonv1.NewStructuredValues(namespace),
					},
					{
						Name:  "APPLICATION",
						Value: *tektonv1.NewStructuredValues("myapp"),
					},
					{
						Name:  "IMAGE_DIGEST",
						Value: *tektonv1.NewStructuredValues("image_digest_value"),
					},
					{
						Name:  "IMAGE_URL",
						Value: *tektonv1.NewStructuredValues("image"),
					},
					{
						Name:  "CHAINS-GIT_URL",
						Value: *tektonv1.NewStructuredValues("git_url_value"),
					},
					{
						Name:  "CHAINS-GIT_COMMIT",
						Value: *tektonv1.NewStructuredValues("git_commit_value"),
					},
				}
				expectedResult, err := json.Marshal(result)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(expectedResult))
			})
		})
	})
	Describe("Pull_request Pipelinerun; Not Ended; without finalizer, annotation, results and application name", func() {
		testFalsePipelineRun = &tektonv1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testPullRequestPipelineRun",
				Namespace: namespace,
				Labels: map[string]string{
					PipelineRunTypeLabel: "pull_request",
				},
				Finalizers: []string{
					"test.appstudio.openshift.io/pipelinerun",
				},
				Annotations: map[string]string{
					"dummy_annotation": "false",
				},
			},
			Spec: tektonv1.PipelineRunSpec{
				PipelineRef: &tektonv1.PipelineRef{},
			},
			Status: tektonv1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: time.Now()},
					CompletionTime: &metav1.Time{Time: time.Now().Add(5 * time.Minute)},
				},
				Status: v1.Status{
					Conditions: v1.Conditions{
						apis.Condition{
							Message: "Tasks Completed: 3 (Failed: 0, Cancelled 0), Incomplete: 10, Skipped:1",
							Reason:  "Running",
							Status:  "Unknown",
							Type:    apis.ConditionSucceeded,
						},
					},
				},
			},
		}
		Context("when a pull_request pipelinerun is created, Still running and has no results", func() {
			It("ShouldProcessPipelineRun should return false", func() {
				got := ShouldProcessPipelineRun(testFalsePipelineRun)
				Expect(got).To(BeFalse())
			})
			It("IsAnnotationExistInPipelineRun shouldreturn false", func() {
				got := IsAnnotationExistInPipelineRun(testFalsePipelineRun, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
				Expect(got).To(BeFalse())
			})
			It("IsFinalizerExistInPipelineRun should return false", func() {
				got := IsFinalizerExistInPipelineRun(testFalsePipelineRun, NotificationPipelineRunFinalizer)
				Expect(got).To(BeFalse())
			})
			It("IsPipelineRunEnded should return false", func() {
				got := IsPipelineRunEnded(testFalsePipelineRun)
				Expect(got).To(BeFalse())
			})
			It("IsPipelineRunEndedSuccessfully should return false", func() {
				got := IsPipelineRunEndedSuccessfully(testFalsePipelineRun)
				Expect(got).To(BeFalse())
			})
			It("GetApplicationNameFromPipelineRun should return empty string", func() {
				got := GetApplicationNameFromPipelineRun(testFalsePipelineRun)
				Expect(got).To(Equal(""))
			})
			It("GetResultsFromPipelineRun should add pipelinerun, namespace and application name to empty results", func() {
				got, err := GetResultsFromPipelineRun(testFalsePipelineRun)
				Expect(err).ToNot(HaveOccurred())
				result := []tektonv1.PipelineRunResult{
					{
						Name:  "PIPELINERUN_NAME",
						Value: *tektonv1.NewStructuredValues("testPullRequestPipelineRun"),
					},
					{
						Name:  "NAMESPACE",
						Value: *tektonv1.NewStructuredValues(namespace),
					},
					{
						Name:  "APPLICATION",
						Value: *tektonv1.NewStructuredValues(""),
					},
				}
				expectedResult, err := json.Marshal(result)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(expectedResult))
			})
		})
	})
	Describe("Not a pipelinerun object", func() {
		notPipelineRun = &tektonv1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-taskrun-pass",
				Namespace: namespace,
			},
			Spec: tektonv1.TaskRunSpec{
				TaskRef: &tektonv1.TaskRef{
					Name: "test-taskrun-pass",
					ResolverRef: tektonv1.ResolverRef{
						Resolver: "bundle",
						Params: tektonv1.Params{
							{Name: "bundle",
								Value: tektonv1.ParamValue{Type: "string", StringVal: "test/example:test"},
							},
							{Name: "name",
								Value: tektonv1.ParamValue{Type: "string", StringVal: "test-task"},
							},
						},
					},
				},
			},
		}
		Context("When non pipelinerun object created", func() {
			It("ShouldProcessPipelineRun should return false", func() {
				got := ShouldProcessPipelineRun(notPipelineRun)
				Expect(got).To(BeFalse())
			})
			It("IsAnnotationExistInPipelineRun should return false", func() {
				got := IsAnnotationExistInPipelineRun(notPipelineRun, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
				Expect(got).To(BeFalse())
			})
			It("IsFinalizerExistInPipelineRun should return false", func() {
				got := IsFinalizerExistInPipelineRun(notPipelineRun, NotificationPipelineRunFinalizer)
				Expect(got).To(BeFalse())
			})
			It("IsPipelineRunEnded should return false", func() {
				got := IsPipelineRunEnded(notPipelineRun)
				Expect(got).To(BeFalse())
			})
			It("IsPipelineRunEndedSuccessfully should return false", func() {
				got := IsPipelineRunEndedSuccessfully(notPipelineRun)
				Expect(got).To(BeFalse())
			})
		})
	})
	Describe("When results fails to marshal", func() {
		testErrMmarshelPipelineRun = &tektonv1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pushPipelineRunName,
				Namespace: notPipelineRun.Namespace,
				Labels: map[string]string{
					PipelineRunTypeLabel: PushPipelineRunTypeValue,
				},
				Finalizers: []string{
					NotificationPipelineRunFinalizer,
				},
				Annotations: map[string]string{
					NotificationPipelineRunAnnotation: NotificationPipelineRunAnnotationValue,
				},
			},
			Spec: tektonv1.PipelineRunSpec{
				PipelineRef: &tektonv1.PipelineRef{},
			},
			Status: tektonv1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					StartTime:      &metav1.Time{Time: time.Now()},
					CompletionTime: &metav1.Time{Time: time.Now().Add(5 * time.Minute)},
					Results: []tektonv1.PipelineRunResult{
						{},
						{
							Name:  "wrong key and value",
							Value: tektonv1.ParamValue{Type: "test"},
						},
					},
				},
				Status: v1.Status{
					Conditions: v1.Conditions{
						apis.Condition{
							Message: "Tasks Completed: 12 (Failed: 0, Cancelled 0), Skipped: 2",
							Reason:  "Completed",
							Status:  "True",
							Type:    apis.ConditionSucceeded,
						},
					},
				},
			},
		}
		Context("when a push pipelinerun is created, and results cannot be marshaled", func() {
			It("GetResultsFromPipelineRun should return an error", func() {
				got, err := GetResultsFromPipelineRun(testErrMmarshelPipelineRun)
				Expect(got).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to marshel results from pipelinerun %s", testErrMmarshelPipelineRun.Name))
			})
		})
	})

	Describe("Successfully adding/removing resources when not exist", func() {
		BeforeEach(func() {
			// Create a pull_request pipelinerun
			testResourcesPipelineRun = &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourcesPipelineRunName,
					Namespace: namespace,
					Labels: map[string]string{
						"pipelines.openshift.io/used-by":     "build-cloud",
						"pipelines.openshift.io/runtime":     "nodejs",
						"pipelines.openshift.io/strategy":    "s2i",
						"appstudio.openshift.io/component":   "component-sample",
						"appstudio.openshift.io/application": "aaa",
					},
				},
				Spec: tektonv1.PipelineRunSpec{
					PipelineRef: &tektonv1.PipelineRef{},
				},
				Status: tektonv1.PipelineRunStatus{
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						StartTime:      &metav1.Time{Time: time.Now()},
						CompletionTime: &metav1.Time{Time: time.Now().Add(5 * time.Minute)},
					},
					Status: v1.Status{
						Conditions: v1.Conditions{
							apis.Condition{
								Message: "Tasks Completed: 3 (Failed: 0, Cancelled 0), Incomplete: 10, Skipped:1",
								Reason:  "Running",
								Status:  "Unknown",
								Type:    apis.ConditionSucceeded,
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, testResourcesPipelineRun)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, testPipelineLookupKey, testPipelineRun)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Finalizer and annotation should not exists
			Expect(controllerutil.ContainsFinalizer(testPipelineRun, NotificationPipelineRunFinalizer)).To(BeFalse())
			Expect(metadata.HasAnnotationWithValue(testPipelineRun, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)).To(BeFalse())
		})
		Context("Pipelinerun without finalizer and annotation", func() {
			It("AddFinalizerToPipelineRun should succeed", func() {
				err := AddFinalizerToPipelineRun(ctx, testPipelineRun, nsr, NotificationPipelineRunFinalizer)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, testPipelineLookupKey, testPipelineRun)
					Expect(err).ToNot(HaveOccurred())
					return controllerutil.ContainsFinalizer(testPipelineRun, NotificationPipelineRunFinalizer)
				}, timeout, interval).Should(BeTrue())
			})
			It("AddAnnotationToPipelineRun should succeed", func() {
				err := AddAnnotationToPipelineRun(ctx, testPipelineRun, nsr, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, testPipelineLookupKey, testPipelineRun)
					Expect(err).ToNot(HaveOccurred())
					return metadata.HasAnnotationWithValue(testPipelineRun, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
				}, timeout, interval).Should(BeTrue())
			})
			It("RemoveFinalizerFromPipelineRun should succeed if finalizer does not exist", func() {
				err := RemoveFinalizerFromPipelineRun(ctx, testPipelineRun, nsr, NotificationPipelineRunFinalizer)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, testPipelineLookupKey, testPipelineRun)
					Expect(err).ToNot(HaveOccurred())
					return !controllerutil.ContainsFinalizer(testPipelineRun, NotificationPipelineRunFinalizer)
				}, timeout, interval).Should(BeTrue())
			})
		})
		AfterEach(func() {
			err := k8sClient.Delete(ctx, testPipelineRun)
			Expect(err == nil || errors.IsNotFound(err)).To(BeTrue())
		})
	})
	Describe("Successfully adding/removing resources when exist", func() {
		BeforeEach(func() {
			// Create a pull_request pipelinerun
			testResourcesPipelineRun = &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourcesPipelineRunName,
					Namespace: namespace,
					Labels: map[string]string{
						"pipelines.openshift.io/used-by":     "build-cloud",
						"pipelines.openshift.io/runtime":     "nodejs",
						"pipelines.openshift.io/strategy":    "s2i",
						"appstudio.openshift.io/component":   "component-sample",
						"appstudio.openshift.io/application": "aaa",
					},
					Finalizers: []string{
						NotificationPipelineRunFinalizer,
					},
					Annotations: map[string]string{
						NotificationPipelineRunAnnotation: NotificationPipelineRunAnnotationValue,
					},
				},
				Spec: tektonv1.PipelineRunSpec{
					PipelineRef: &tektonv1.PipelineRef{},
				},
				Status: tektonv1.PipelineRunStatus{
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						StartTime:      &metav1.Time{Time: time.Now()},
						CompletionTime: &metav1.Time{Time: time.Now().Add(5 * time.Minute)},
					},
					Status: v1.Status{
						Conditions: v1.Conditions{
							apis.Condition{
								Message: "Tasks Completed: 3 (Failed: 0, Cancelled 0), Incomplete: 10, Skipped:1",
								Reason:  "Running",
								Status:  "Unknown",
								Type:    apis.ConditionSucceeded,
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, testResourcesPipelineRun)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, testPipelineLookupKey, testPipelineRun)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Finalizer and annotation should exist
			Expect(controllerutil.ContainsFinalizer(testPipelineRun, NotificationPipelineRunFinalizer)).To(BeTrue())
			Expect(metadata.HasAnnotationWithValue(testPipelineRun, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)).To(BeTrue())
		})
		Context("Pipelinerun with finalizer and annotation", func() {
			It("AddFinalizerToPipelineRun should succeed if finalizer exists", func() {
				err := AddFinalizerToPipelineRun(ctx, testPipelineRun, nsr, NotificationPipelineRunFinalizer)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, testPipelineLookupKey, testPipelineRun)
					Expect(err).ToNot(HaveOccurred())
					return controllerutil.ContainsFinalizer(testPipelineRun, NotificationPipelineRunFinalizer)
				}, timeout, interval).Should(BeTrue())
			})
			It("AddAnnotationToPipelineRun should succeed if annotation exists", func() {
				err := AddAnnotationToPipelineRun(ctx, testPipelineRun, nsr, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, testPipelineLookupKey, testPipelineRun)
					Expect(err).ToNot(HaveOccurred())
					return metadata.HasAnnotationWithValue(testPipelineRun, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
				}, timeout, interval).Should(BeTrue())
			})
			It("RemoveFinalizerFromPipelineRun should succeed", func() {
				err := RemoveFinalizerFromPipelineRun(ctx, testPipelineRun, nsr, NotificationPipelineRunFinalizer)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, testPipelineLookupKey, testPipelineRun)
					Expect(err).ToNot(HaveOccurred())
					return !controllerutil.ContainsFinalizer(testPipelineRun, NotificationPipelineRunFinalizer)
				}, timeout, interval).Should(BeTrue())
			})
		})
		AfterEach(func() {
			err := k8sClient.Delete(ctx, testPipelineRun)
			Expect(err == nil || errors.IsNotFound(err)).To(BeTrue())
		})
	})
	Describe("Test errors", func() {
		Context("When Patch action returns an error", func() {
			It("AddFinalizerToPipelineRun should return an error", func() {
				err := AddFinalizerToPipelineRun(ctx, testPipelineRun, fakeErrorNsr, NotificationPipelineRunFinalizer)
				Expect(err).To(HaveOccurred())
			})
			It("AddAnnotationToPipelineRun should return an error", func() {
				err := AddAnnotationToPipelineRun(ctx, testPipelineRun, fakeErrorNsr, NotificationPipelineRunAnnotation, NotificationPipelineRunAnnotationValue)
				Expect(err).To(HaveOccurred())
			})
			It("RemoveFinalizerFromPipelineRun should return an error", func() {
				err := RemoveFinalizerFromPipelineRun(ctx, testPipelineRun, fakeErrorNsr, NotificationPipelineRunFinalizer)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ShouldProcessPipelineRun filtering tests", func() {
		var (
			pushPipelineRun,
			annotatedPipelineRun,
			noPipelineRun *tektonv1.PipelineRun
		)

		BeforeEach(func() {
			// PipelineRun with push label (default case)
			pushPipelineRun = &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "push-pipeline",
					Namespace: namespace,
					Labels: map[string]string{
						PipelineRunTypeLabel: PushPipelineRunTypeValue,
					},
				},
			}

			// PipelineRun with custom-build-trigger annotation
			annotatedPipelineRun = &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "annotated-pipeline",
					Namespace: namespace,
					Annotations: map[string]string{
						"custom-build-trigger": "https://ci.example.com/build/123",
					},
				},
			}

			// PipelineRun with no relevant labels or annotations
			noPipelineRun = &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-filter-pipeline",
					Namespace: namespace,
				},
			}
		})

		Context("Default filtering (no env vars set)", func() {
			It("should process push PipelineRuns", func() {
				Expect(ShouldProcessPipelineRun(pushPipelineRun)).To(BeTrue())
			})

			It("should not process annotated PipelineRuns without push label", func() {
				Expect(ShouldProcessPipelineRun(annotatedPipelineRun)).To(BeFalse())
			})

			It("should not process PipelineRuns without event-type label", func() {
				Expect(ShouldProcessPipelineRun(noPipelineRun)).To(BeFalse())
			})
		})

		Context("Label filtering with NOTIFICATION_FILTER_LABELS", func() {
			It("should process push PipelineRuns when filter matches", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_LABELS", "pipelinesascode.tekton.dev/event-type=push")
				Expect(ShouldProcessPipelineRun(pushPipelineRun)).To(BeTrue())
			})

			It("should not process PipelineRuns without matching label", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_LABELS", "pipelinesascode.tekton.dev/event-type=push")
				Expect(ShouldProcessPipelineRun(annotatedPipelineRun)).To(BeFalse())
			})
		})

		Context("Annotation filtering with NOTIFICATION_FILTER_ANNOTATIONS", func() {
			It("should process PipelineRuns when annotation filter matches", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_ANNOTATIONS", "custom-build-trigger")
				Expect(ShouldProcessPipelineRun(annotatedPipelineRun)).To(BeTrue())
			})

			It("should not process PipelineRuns without matching annotation", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_ANNOTATIONS", "custom-build-trigger")
				Expect(ShouldProcessPipelineRun(pushPipelineRun)).To(BeFalse())
			})

			It("should handle multiple annotation filters", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_ANNOTATIONS", "custom-build-trigger,custom-annotation")
				Expect(ShouldProcessPipelineRun(annotatedPipelineRun)).To(BeTrue())
			})
		})

		Context("Combined label and annotation filtering", func() {
			It("should process PipelineRuns matching either label or annotation", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_LABELS", "pipelinesascode.tekton.dev/event-type=push")
				GinkgoT().Setenv("NOTIFICATION_FILTER_ANNOTATIONS", "custom-build-trigger")

				Expect(ShouldProcessPipelineRun(pushPipelineRun)).To(BeTrue())
				Expect(ShouldProcessPipelineRun(annotatedPipelineRun)).To(BeTrue())
			})

			It("should not process PipelineRuns matching neither filter", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_LABELS", "pipelinesascode.tekton.dev/event-type=push")
				GinkgoT().Setenv("NOTIFICATION_FILTER_ANNOTATIONS", "custom-build-trigger")

				Expect(ShouldProcessPipelineRun(noPipelineRun)).To(BeFalse())
			})
		})

		Context("Edge cases", func() {
			It("should return false for non-PipelineRun objects", func() {
				taskRun := &tektonv1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-taskrun",
					},
				}
				Expect(ShouldProcessPipelineRun(taskRun)).To(BeFalse())
			})

			It("should handle empty filter values gracefully", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_LABELS", "")
				GinkgoT().Setenv("NOTIFICATION_FILTER_ANNOTATIONS", "")
				Expect(ShouldProcessPipelineRun(pushPipelineRun)).To(BeTrue())
			})

			It("should handle whitespace in filter configuration", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_LABELS", " pipelinesascode.tekton.dev/event-type=push ")
				Expect(ShouldProcessPipelineRun(pushPipelineRun)).To(BeTrue())
			})

			It("should return false when label filter has no '=' separator", func() {
				GinkgoT().Setenv("NOTIFICATION_FILTER_LABELS", "badfilter")
				Expect(ShouldProcessPipelineRun(pushPipelineRun)).To(BeFalse())
			})
		})
	})
})
