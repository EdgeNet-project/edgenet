package multitenancy

import (
	"context"

	v1 "github.com/ubombar/edgenet-kubebuilder/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This interface contains the necessary functions to perform the operations related to
// multi tenancy. Most of the implementation here is retrieved from the old implementation.
// However, some of the functions are changed.
type MultiTenancyManager interface {
	// Remove the tenant and all of the other artifacts created with it including,
	// subtenants, subnamespaces etc.
	TenantCleanup(context.Context, *v1.Tenant) error

	// Creates a core namespace (same name with the tenant) and sets the resource allocation.
	// Returns nil, if the namespace already exists.
	CreateCoreNamespace(context.Context, *v1.Tenant) error
}

type multiTenancyManager struct {
	MultiTenancyManager
	client client.Client
}

func NewMultiTenancyManager(ctx context.Context, client client.Client) (MultiTenancyManager, error) {
	return &multiTenancyManager{
		client: client,
	}, nil
}

func (m *multiTenancyManager) TenantCleanup(ctx context.Context, t *v1.Tenant) error {
	// Get the corenamespace name
	coreNamespaceName := ResolveCoreNamespaceName(t.Name)
	coreNamespaceObjectKey := client.ObjectKey{Name: coreNamespaceName}

	// Create the namespace
	coreNamespace := corev1.Namespace{}
	err := m.client.Get(ctx, coreNamespaceObjectKey, &coreNamespace)

	if err != nil {
		return err
	}

	return m.client.Delete(ctx, &coreNamespace)
}

func (m *multiTenancyManager) CreateCoreNamespace(ctx context.Context, t *v1.Tenant) error {
	// Get the corenamespace name
	coreNamespaceName := ResolveCoreNamespaceName(t.Name)
	coreNamespaceObjectKey := client.ObjectKey{Name: coreNamespaceName}

	// Create the namespace
	coreNamespace := corev1.Namespace{}
	err := m.client.Get(ctx, coreNamespaceObjectKey, &coreNamespace)

	// If there is no core namespace, create one
	if err != nil && errors.IsNotFound(err) {
		// Create the namespace here
		coreNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: coreNamespaceName,
				Labels: map[string]string{
					"edge-net.io/tenant": t.Name,
				},
			},
		}
		// Create the core namespace
		err := m.client.Create(ctx, &coreNamespace)

		// If there is a creation error
		if err != nil {
			return err
		}
	} else if err != nil { // If there is another error, just return
		return err
	}

	// Now allocate resources for specified as in the initial request

	return nil
}
