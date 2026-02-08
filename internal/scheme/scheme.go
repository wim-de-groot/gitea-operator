package scheme

import (
	alphav1 "github.com/wim-de-groot/gitea-operator/api/alphav1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// Builder contains the fluent methods to build a schema
type Builder struct {
	scheme *runtime.Scheme
}

// New creates a new builder
func New() *Builder {
	return &Builder{scheme: runtime.NewScheme()}
}

// WithClientGoScheme adds the kubernetes/scheme
func (b *Builder) WithClientGoScheme() *Builder {
	_ = clientgoscheme.AddToScheme(b.scheme)

	return b
}

// WithAPIV1 adds the v1 scheme
func (b *Builder) WithAlphaV1() *Builder {
	_ = alphav1.AddToScheme(b.scheme)

	return b
}

// WithAPIExtensionV1 adds apiextensions/v1
func (b *Builder) WithAPIExtensionV1() *Builder {
	_ = apiextensionsv1.AddToScheme(b.scheme)

	return b
}

// Build returns the built scheme
func (b *Builder) Build() *runtime.Scheme {
	return b.scheme
}

// BuildWithAllKnownScheme registers all the API used by the manager
func BuildWithAllKnownScheme() *runtime.Scheme {
	return New().
		WithAlphaV1().
		WithClientGoScheme().
		WithAPIExtensionV1().
		Build()

	// +kubebuilder:scaffold:scheme
}
