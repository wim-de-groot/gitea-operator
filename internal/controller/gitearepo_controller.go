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
	"time"

	g "code.gitea.io/sdk/gitea"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	alphaV1 "github.com/wim-de-groot/gitea-operator/api/alphav1"
	"github.com/wim-de-groot/gitea-operator/internal/management/utils"
)

const finalizerName = "core.gitea.com/finalizer"

type GiteaRepoReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=core.gitea.com,resources=gitearepoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.gitea.com,resources=gitearepoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.gitea.com,resources=gitearepoes/finalizers,verbs=update
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

func (r *GiteaRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	repo := &alphaV1.GiteaRepo{}
	if err := r.Client.Get(ctx, req.NamespacedName, repo); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, fmt.Errorf("unable to get gitea instance: %w", err)
		}
	}

	gitea := &alphaV1.Gitea{}
	giteaNamespacedName := types.NamespacedName{Name: repo.Spec.GiteaRef.Name, Namespace: repo.Spec.GiteaRef.Namespace}
	if err := r.Client.Get(ctx, giteaNamespacedName, gitea); err != nil {
		if errors.IsNotFound(err) {
			r.Recorder.Eventf(
				gitea,
				nil,
				coreV1.EventTypeWarning,
				"Degrated",
				"GiteaNotFound",
				fmt.Sprintf("Gitea instance '%s' not found in namespace '%s'", repo.Spec.GiteaRef.Name, repo.Spec.GiteaRef.Namespace),
				repo.Name,
				repo.Namespace,
			)
			log.Info("Gitea instance %s not found in namespace %s", repo.Spec.GiteaRef.Name, repo.Spec.GiteaRef.Namespace)
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, fmt.Errorf("unable to get gitea instance: %w", err)
		}
	}

	secret := &coreV1.Secret{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: gitea.Spec.CredentialsSecretName, Namespace: gitea.Namespace}, secret); err != nil {
		if errors.IsNotFound(err) {
			r.Recorder.Eventf(
				gitea,
				secret,
				coreV1.EventTypeWarning,
				"Degrated",
				"GiteaCredentialsNotFound",
				fmt.Sprintf(
					"Credentials '%s' for Gitea instance '%s' not found in namespace '%s'",
					gitea.Spec.CredentialsSecretName,
					gitea.Name,
					gitea.Namespace,
				),
				repo.Name,
				repo.Namespace,
			)
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, fmt.Errorf("unable to get secret: %s, because of: %w", gitea.Spec.CredentialsSecretName, err)
		}
	}

	username, password, err := utils.GetUsernameAndPasswordFromSecret(secret, gitea.Spec.UsernameSecretKey, gitea.Spec.PasswordSecretKey)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to get username and password from secret, because of %w", err)
	}

	giteaClient, err := g.NewClient(gitea.Spec.Url, g.SetBasicAuth(username, password))

	if repo.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(repo, finalizerName) {
			controllerutil.AddFinalizer(repo, finalizerName)
			if err := r.Update(ctx, repo); err != nil {
				return ctrl.Result{}, nil
			}
		}

		_, resp, err := giteaClient.GetRepo(username, repo.Name)
		if err != nil && resp.StatusCode != 404 {
			return ctrl.Result{}, fmt.Errorf("unable to validate if repo exists, because of %w", err)
		}

		if err != nil && resp.StatusCode == 404 {
			log.Info("Repo not found, creating...")
			_, _, err := giteaClient.CreateRepo(g.CreateRepoOption{
				Name:          repo.Name,
				DefaultBranch: "main",
			})

			if err != nil {
				log.Error(err, "repo %s could not be created", repo.Name)
				meta.SetStatusCondition(&repo.Status.Conditions, v1.Condition{
					Type:    "Degraded",
					Status:  v1.ConditionFalse,
					Reason:  "GiteaRepoCreationFailed",
					Message: "Gitea repository been created",
				})
			}

			meta.SetStatusCondition(&repo.Status.Conditions, v1.Condition{
				Type:    "Available",
				Status:  v1.ConditionTrue,
				Reason:  "GiteaRepoCreated",
				Message: fmt.Sprintf("Gitea repository %s been created", repo.Name),
			})
		} else {
			meta.SetStatusCondition(&repo.Status.Conditions, v1.Condition{
				Type:    "Available",
				Status:  v1.ConditionTrue,
				Reason:  "GiteaRepoAvailable",
				Message: fmt.Sprintf("Gitea repository %s available", repo.Name),
			})
		}

		if err := r.Status().Update(ctx, repo); err != nil {
			log.Error(err, "unable to update GiteaRepo status")
			return ctrl.Result{}, err
		}
	} else {
		if controllerutil.ContainsFinalizer(repo, finalizerName) {

			meta.SetStatusCondition(&repo.Status.Conditions, v1.Condition{
				Type:    "Available",
				Status:  v1.ConditionTrue,
				Reason:  "GiteaRepoTerminating",
				Message: fmt.Sprintf("Gitea repository %s is being deleted", repo.Name),
			})

			if _, err := giteaClient.DeleteRepo(username, repo.Name); err != nil {
				meta.SetStatusCondition(&repo.Status.Conditions, v1.Condition{
					Type:    "Degraded",
					Status:  v1.ConditionTrue,
					Reason:  "GiteaRepoNotTerminating",
					Message: fmt.Sprintf("Gitea repository %s can not be deleted because of unexpected error, please check the logs for more information", repo.Name),
				})

				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(repo, finalizerName)
			if err := r.Update(ctx, repo); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GiteaRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&alphaV1.GiteaRepo{}).
		Named("gitearepo").
		Complete(r)
}
