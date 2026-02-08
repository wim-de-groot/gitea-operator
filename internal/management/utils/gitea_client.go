package utils

import (
	"code.gitea.io/sdk/gitea"
	corealphav1 "github.com/wim-de-groot/gitea-operator/api/alphav1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateGiteaClient(gitea *corealphav1.Gitea, client client.Client) {

	if err := client.Get(ctx, req.NamespacedName, gitea); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Instance not found")
		}

		return ctrl.Result{}, fmt.Errorf("unable to get gitea instance: %w", err)
	}

	log.Info(fmt.Sprintf("Instance found, name: %s, namespace: %s", gitea.Name, gitea.Namespace))

	secret := &coreV1.Secret{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: gitea.Spec.CredentialsSecretName}, secret); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Secret not found")
		}

		return ctrl.Result{}, fmt.Errorf("unable to get secret: %s, because of: %w", gitea.Spec.CredentialsSecretName, err)
	}

	username, password, err := utils.GetUsernameAndPasswordFromSecret(secret, gitea.Spec.UsernameSecretKey, gitea.Spec.PasswordSecretKey)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to get username and password from secret, because of %w", err)
	}

	_, err = g.NewClient(gitea.Spec.Url, g.SetBasicAuth(username, password))
}
