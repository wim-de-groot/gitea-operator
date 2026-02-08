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

	giteaSdk "code.gitea.io/sdk/gitea"
	alphaV1 "github.com/wim-de-groot/gitea-operator/api/alphav1"
	utils "github.com/wim-de-groot/gitea-operator/internal/management/utils"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const successConnectionRetryPeriod = time.Minute * 30

// GiteaReconciler reconciles a Gitea object
type GiteaReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=core.gitea.com,resources=gitea,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.gitea.com,resources=gitea/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.gitea.com,resources=gitea/finalizers,verbs=update
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

func (r *GiteaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gitea := &alphaV1.Gitea{}

	if err := r.Get(ctx, req.NamespacedName, gitea); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("not able to get gitea instance '%s' because of: %w", req.Name, err)
	}

	nns := types.NamespacedName{Namespace: gitea.Namespace, Name: gitea.Spec.CredentialsSecretName}
	secret := &coreV1.Secret{}
	if err := r.Get(ctx, nns, secret); err != nil {
		if errors.IsNotFound(err) {
			r.Recorder.Eventf(
				gitea,
				secret,
				coreV1.EventTypeWarning,
				"Degraded",
				"GetCredentials",
				"Secret '%s' with credentials for gitea instane '%s' could not be retrieved from namespace '%s'.",
				gitea.Spec.CredentialsSecretName,
				gitea.Name,
				gitea.Namespace,
			)
		}

		return ctrl.Result{}, fmt.Errorf("not able to get secret instance '%s' because of: %w", nns.Name, err)
	}

	username, password, err := utils.GetUsernameAndPasswordFromSecret(secret, gitea.Spec.UsernameSecretKey, gitea.Spec.PasswordSecretKey)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to get username and password from secret '%s', because of %w", secret.Name, err)
	}

	basicAuthOpts := giteaSdk.SetBasicAuth(username, password)
	_, err = giteaSdk.NewClient(gitea.Spec.Url, basicAuthOpts)

	meta.SetStatusCondition(&gitea.Status.Conditions, v1.Condition{
		Type:    "Connnecting",
		Status:  v1.ConditionUnknown,
		Reason:  "GiteaConnecting",
		Message: "Connecting to gitea.",
	})

	if err != nil {
		meta.SetStatusCondition(&gitea.Status.Conditions, v1.Condition{
			Type:    "Degraded",
			Status:  v1.ConditionFalse,
			Reason:  "GiteaConnectionDegraded",
			Message: fmt.Sprintf("Failed to connect to Gitea instance '%s' in namespace '%s'.", gitea.Name, gitea.Namespace),
		})
	}

	meta.SetStatusCondition(&gitea.Status.Conditions, v1.Condition{
		Type:    "Available",
		Status:  v1.ConditionUnknown,
		Reason:  "GiteaConnectionAvailable",
		Message: fmt.Sprintf("Successfully connected to Gitea instance '%s' in namespace '%s'.", gitea.Name, gitea.Namespace),
	})

	err = r.Status().Update(ctx, gitea)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to create gitea client because of: %w", err)
	}

	return ctrl.Result{RequeueAfter: successConnectionRetryPeriod}, nil
}

func (r *GiteaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&alphaV1.Gitea{}).
		Named("gitea").
		Complete(r)
}
