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
	"errors"
	"fmt"
	"go/build"
	"path/filepath"
	"runtime"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	toolkit "github.com/konflux-ci/operator-toolkit/test"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var k8sManager manager.Manager
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc
var mn, fakeErrorNotify *MockNotifier
var nsr, fakeErrorNotifyNsr, fakeErrorNsr *NotificationServiceReconciler

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

type MockNotifier struct {
	Counter            int
	shouldThroughError bool
}

func (mn *MockNotifier) Notify(ctx context.Context, message string) error {
	mn.Counter++
	if mn.shouldThroughError {
		return errors.New("Failed to Notify")
	}
	return nil
}

type clientMock struct {
	client.Client
	shouldThroughError bool
}

func (p *clientMock) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if p.shouldThroughError {
		return errors.New("Failed to patch")
	}
	return nil
}

var _ = BeforeEach(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(
				build.Default.GOPATH,
				"pkg", "mod", toolkit.GetRelativeDependencyPath("tektoncd/pipeline"), "config",
			),
			filepath.Join(
				build.Default.GOPATH,
				"pkg", "mod", toolkit.GetRelativeDependencyPath("tektoncd/pipeline"), "config", "300-crds",
			),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Adding tektonv1 to the scheme and validate it
	Expect(tektonv1.AddToScheme(clientsetscheme.Scheme)).To(Succeed())

	// +kubebuilder:scaffold:scheme

	// Disabling Cache for pipelinerun to avoid reflector watch errors
	// https://github.com/kubernetes-sigs/controller-runtime/issues/2723
	k8sClient, err = client.New(cfg, client.Options{
		Scheme: clientsetscheme.Scheme,
		Cache: &client.CacheOptions{
			DisableFor: []client.Object{
				&tektonv1.PipelineRun{},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	/*
		Create k8sManager and validate it
		Note that we set up both a "live" k8s client and a separate client from the manager. This is because when making
		assertions in tests, you generally want to assert against the live state of the API server. If you use the client
		from the manager (`k8sManager.GetClient`), you'd end up asserting against the contents of the cache instead, which is
		slower and can introduce flakiness into your tests. We could use the manager's `APIReader` to accomplish the same
		thing, but that would leave us with two clients in our test assertions and setup (one for reading, one for writing),
		and it'd be easy to make mistakes.
		https://github.com/kubernetes-sigs/kubebuilder/blob/de1cc60900b896b2195e403a40c976a892df4921/docs/book/src/cronjob-tutorial/testdata/project/internal/controller/suite_test.go#L136
	*/
	// Skip controller name validation for the unit tests
	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: clientsetscheme.Scheme,
		Controller: config.Controller{
			SkipNameValidation: &[]bool{true}[0],
		},
	})

	Expect(err).ToNot(HaveOccurred())
	// Create a mock of notifier
	mn = &MockNotifier{shouldThroughError: false}
	nsr = &NotificationServiceReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Log:      k8sManager.GetLogger(),
		Notifier: mn,
	}
	// err = nsr.SetupWithManager(k8sManager)
	// Expect(err).ToNot(HaveOccurred())
	fakeErrorNotify = &MockNotifier{shouldThroughError: true}
	fakeErrorNotifyNsr = &NotificationServiceReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Log:      k8sManager.GetLogger(),
		Notifier: fakeErrorNotify,
	}
	fakeErrorPatchClient := &clientMock{shouldThroughError: true}
	fakeErrorNsr = &NotificationServiceReconciler{
		Client:   fakeErrorPatchClient,
		Scheme:   k8sManager.GetScheme(),
		Log:      k8sManager.GetLogger(),
		Notifier: mn,
	}

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

})

var _ = AfterEach(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
