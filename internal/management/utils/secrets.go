package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func GetUsernameAndPasswordFromSecret(secret *corev1.Secret, usernameSecretKey string, passwordSecretKey string) (string, string, error) {
	if _, ok := secret.Data[usernameSecretKey]; !ok {
		return "", "", fmt.Errorf("%s doesn't exist in secret %s", usernameSecretKey, secret.Name)
	}

	if _, ok := secret.Data[passwordSecretKey]; !ok {
		return "", "", fmt.Errorf("%s doesn't exist in secret %s", passwordSecretKey, secret.Name)
	}

	return string(secret.Data[usernameSecretKey]), string(secret.Data[passwordSecretKey]), nil
}
