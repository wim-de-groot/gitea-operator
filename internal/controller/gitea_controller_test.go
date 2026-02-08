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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	corealphav1 "github.com/wim-de-groot/gitea-operator/api/alphav1"
)

var _ = Describe("Gitea Controller", func() {
	var env *testingEnvironment
	BeforeEach(func() {
		env = buildTestEnvironment()
	})

	Context("Gitea Controller", func() {
		var namespace *corev1.Namespace
		var secret *corev1.Secret
		var gitea *corealphav1.Gitea

		BeforeEach(func(ctx context.Context) {
			namespace = newFakeNamespace(env.client)
			secret = newFakeSecret(env.client, namespace)

			By("creating the custom resource for the Kind Gitea")
			gitea = env.newFakeGitea(secret)
		})

		It("should successfully reconcile Gitea", func() {
			By("checking if condtions have been added")
			Eventually(func(g Gomega) {
				g.Expect(gitea.Status.Conditions).To(BeEmpty())
			}).Should(Succeed())
		})
	})
})
