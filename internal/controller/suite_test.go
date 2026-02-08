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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	corealphav1 "github.com/wim-de-groot/gitea-operator/api/alphav1"

	schemeBuilder "github.com/wim-de-groot/gitea-operator/internal/scheme"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"k8s.io/apimachinery/pkg/util/rand"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	err = corealphav1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}

type testingEnvironment struct {
	client              client.WithWatch
	giteaReconciler     GiteaReconciler
	giteaRepoReconciler GiteaRepoReconciler
	scheme              *runtime.Scheme
}

func buildTestEnvironment() *testingEnvironment {
	var err error
	Expect(err).ToNot(HaveOccurred())

	s := schemeBuilder.BuildWithAllKnownScheme()
	k8sClient := fake.
		NewClientBuilder().
		WithScheme(s).
		WithStatusSubresource(&corealphav1.Gitea{}, &corealphav1.GiteaRepo{}, &corev1.Secret{}).
		Build()

	Expect(err).ToNot(HaveOccurred())

	giteaReconciler := GiteaReconciler{
		Client:   k8sClient,
		Scheme:   s,
		Recorder: events.NewFakeRecorder(120),
	}

	giteaRepoReconciler := GiteaRepoReconciler{
		Client:   k8sClient,
		Scheme:   s,
		Recorder: events.NewFakeRecorder(120),
	}

	Expect(err).ToNot(HaveOccurred())

	return &testingEnvironment{
		k8sClient,
		giteaReconciler,
		giteaRepoReconciler,
		s,
	}
}

func newFakeNamespace(k8sClient client.Client) *corev1.Namespace {
	name := rand.String(10)

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	return namespace
}

func newFakeSecret(k8sClient client.Client, namespace *corev1.Namespace) *corev1.Secret {
	name := rand.String(10)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace.Name,
		},
		StringData: map[string]string{
			"username": rand.String(10),
			"password": rand.String(10),
		},
	}

	err := k8sClient.Create(context.Background(), secret)
	Expect(err).ToNot(HaveOccurred())

	return secret
}

func (env *testingEnvironment) newFakeGitea(secret *corev1.Secret) *corealphav1.Gitea {
	name := rand.String(10)
	urlPattern := "%s.%s.svc.cluster.local"
	giteaUrl := fmt.Sprintf(urlPattern, name, secret.Namespace)

	gitea := &corealphav1.Gitea{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: secret.Namespace,
		},
		Spec: corealphav1.GiteaSpec{
			Url:                   giteaUrl,
			CredentialsSecretName: secret.Namespace,
			UsernameSecretKey:     "username",
			PasswordSecretKey:     "password",
		},
	}

	err := env.giteaReconciler.Create(ctx, gitea)
	Expect(err).NotTo(HaveOccurred())

	return gitea
}
