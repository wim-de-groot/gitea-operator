package gitea

import (
	context "context"
	fmt "fmt"

	giteaSDK "code.gitea.io/sdk/gitea"
	alphaV1 "github.com/wim-de-groot/gitea-operator/api/alphav1"
	utils "github.com/wim-de-groot/gitea-operator/internal/management/utils"
	coreV1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Service struct {
	Client *giteaSDK.Client
	gitea  *alphaV1.Gitea
}

func (service *Service) InstanceName() string {
	return service.gitea.Name
}

func (service *Service) UpdateStatus(cli k8sClient.Client, ctx context.Context, connected bool) error {
	service.gitea.Status.Connected = connected

	return cli.Status().Update(ctx, service.gitea)
}

func NewService(cli k8sClient.Client, ctx context.Context, nns types.NamespacedName) (*Service, error) {
	gitea := &alphaV1.Gitea{}
	if err := cli.Get(ctx, nns, gitea); err != nil && errors.IsNotFound(err) {
		return nil, fmt.Errorf("not able to get gitea instance '%s' because of: %w", nns.Name, err)
	}

	nns = types.NamespacedName{Namespace: nns.Namespace, Name: gitea.Spec.CredentialsSecretName}
	secret := &coreV1.Secret{}
	if err := cli.Get(ctx, nns, secret); err != nil && errors.IsNotFound(err) {
		return nil, fmt.Errorf("not able to get secret instance '%s' because of: %w", nns.Name, err)
	}

	username, password, err := utils.GetUsernameAndPasswordFromSecret(secret, gitea.Spec.UsernameSecretKey, gitea.Spec.PasswordSecretKey)
	if err != nil {
		return nil, fmt.Errorf("unable to get username and password from secret '%s', because of %w", secret.Name, err)
	}

	opt := giteaSDK.SetBasicAuth(username, password)
	client, err := giteaSDK.NewClient(gitea.Spec.Url, opt)

	return &Service{Client: client, gitea: gitea}, nil
}
