/*
Copyright 2026.

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Controller RBAC", func() {
	It("denies PipelineRun creation, replacement, and status writes in a tenant namespace", func() {
		pipelineRun := &tektonv1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac-pipelinerun",
				Namespace: "default",
			},
			Spec: tektonv1.PipelineRunSpec{
				PipelineRef: &tektonv1.PipelineRef{Name: "test-pipeline"},
			},
		}
		controllerClient := k8sManager.GetClient()

		By("creating a fixture with the admin identity")
		Expect(k8sClient.Create(ctx, pipelineRun)).To(Succeed())

		By("waiting for the role to authorize metadata patches before checking denials")
		metadataPatch := client.RawPatch(types.MergePatchType, []byte(`{"metadata":{"annotations":{"rbac-test":"allowed"}}}`))
		Eventually(func() error {
			return controllerClient.Patch(ctx, pipelineRun.DeepCopy(), metadataPatch)
		}, 10*time.Second, 250*time.Millisecond).Should(Succeed())

		By("rejecting creation with the controller identity")
		forgedRun := &tektonv1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{Name: "forged-pipelinerun", Namespace: pipelineRun.Namespace},
			Spec:       pipelineRun.Spec,
		}
		Expect(controllerClient.Create(ctx, forgedRun)).To(
			MatchError(apierrors.IsForbidden, "a Forbidden error"))

		By("rejecting replacement of an existing PipelineRun")
		Expect(controllerClient.Update(ctx, pipelineRun.DeepCopy())).To(
			MatchError(apierrors.IsForbidden, "a Forbidden error"))

		By("rejecting status updates and patches")
		Expect(controllerClient.Status().Update(ctx, pipelineRun.DeepCopy())).To(
			MatchError(apierrors.IsForbidden, "a Forbidden error"))
		patch := client.RawPatch(types.MergePatchType, []byte(`{"status":{"conditions":[{"type":"Succeeded","status":"True"}]}}`))
		Expect(controllerClient.Status().Patch(ctx, pipelineRun.DeepCopy(), patch)).To(
			MatchError(apierrors.IsForbidden, "a Forbidden error"))
	})
})
