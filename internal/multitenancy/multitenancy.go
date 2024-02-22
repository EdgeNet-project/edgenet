package multitenancy

import (
	"context"

	v1 "github.com/ubombar/edgenet-kubebuilder/api/v1"
	utils "github.com/ubombar/edgenet-kubebuilder/internal"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	CreateCoreNamespace(context.Context, *v1.Tenant, types.UID) error

	// Same as the CreateCoreNamespace except gets the UID from local cluster.
	CreateCoreNamespaceLocal(ctx context.Context, t *v1.Tenant) error
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
	// The core namespace should automatically deleted because of the owner references.
	// // Get the corenamespace name
	// coreNamespaceName := ResolveCoreNamespaceName(t.Name)
	// coreNamespaceObjectKey := client.ObjectKey{Name: coreNamespaceName}

	// // Create the namespace
	// coreNamespace := corev1.Namespace{}
	// err := m.client.Get(ctx, coreNamespaceObjectKey, &coreNamespace)

	// if err != nil {
	// 	return err
	// }

	// return m.client.Delete(ctx, &coreNamespace)
	return nil
}

// Same as the CreateCoreNamespace except automaticaly populates the cluster UID from the local
// Cluster's kube-system namesapce
func (m *multiTenancyManager) CreateCoreNamespaceLocal(ctx context.Context, t *v1.Tenant) error {
	clusterUID, err := utils.GetClusterUID(ctx, m.client)

	if err != nil {
		return err
	}

	return m.CreateCoreNamespace(ctx, t, clusterUID)
}

// Creates a core namespace, sets the ownership references and does resource allocation.
// The clusterUID is given as a future federation concept.
func (m *multiTenancyManager) CreateCoreNamespace(ctx context.Context, t *v1.Tenant, clusterUID types.UID) error {
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
				// Set the labels
				Labels: map[string]string{
					"edge-net.io/tenant":      t.GetName(),
					"edge-net.io/kind":        "core",
					"edge-net.io/tenant-uid":  string(t.GetUID()),
					"edge-net.io/cluster-uid": string(clusterUID),
				},
				// Set the annotations, for now empty
				Annotations: map[string]string{},
				// Set the owner reference
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(t, t.GroupVersionKind()),
				},
			},
		}
		// Create the core namespace
		err := m.client.Create(ctx, &coreNamespace)

		// If the namespace already exists then update it
		if errors.IsAlreadyExists(err) {
			return m.client.Update(ctx, &coreNamespace)
		}

		// If there is a creation error other than already exsits
		if err != nil {
			return err
		}
	} else if err != nil { // If there is another error, just return
		return err
	}

	// Set the resource quota for the namespace

	return nil
}
