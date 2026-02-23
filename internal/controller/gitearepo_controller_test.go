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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corealphav1 "github.com/wim-de-groot/gitea-operator/api/alphav1"
)

var _ = Describe("GiteaRepo Controller", func() {
	var env *testingEnvironment
	BeforeEach(func() {
		env = buildTestEnvironment()
	})

	Context("When reconciling a resource", func() {
		var namespace *corev1.Namespace
		var secret *corev1.Secret
		var gitea *corealphav1.Gitea
		var giteaRepo *corealphav1.GiteaRepo
		BeforeEach(func(ctx context.Context) {
			namespace = newFakeNamespace(env.client)
			secret = newFakeSecret(env.client, namespace)

			By("creating the custom resource for the Kind Gitea")
			gitea = env.newFakeGitea(secret)

			By("creating the custom resource for the Kind GiteaRepo")
			giteaRepo = &corealphav1.GiteaRepo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "repo-example",
					Namespace: namespace.Name,
				},
				Spec: corealphav1.GiteaRepoSpec{
					GiteaRef: corealphav1.GiteaRef{
						Name:      gitea.Name,
						Namespace: namespace.Name,
					},
				},
			}

			err := env.giteaRepoReconciler.Create(ctx, giteaRepo)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not create repository", func() {
			By("checking if condtions have not been added")
			Eventually(func(g Gomega) {
				g.Expect(giteaRepo.Status.Conditions).To(BeEmpty())
			}).Should(Succeed())
		})
	})
})
