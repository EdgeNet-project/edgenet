package multitenancy

import (
	"context"

	antreav1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	v1 "github.com/ubombar/edgenet-kubebuilder/api/v1"
	"github.com/ubombar/edgenet-kubebuilder/internal/utils"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	// Creates a new tenant role binding with admin priviliages. Requires "edgenet:tenant-admin" role
	// to work.
	CreateTenantAdminRoleBinding(ctx context.Context, t *v1.Tenant) error

	// Create the network policy. If specified creates the cluster network policy as well.
	CreateTenantNetworkPolicy(ctx context.Context, t *v1.Tenant) error
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
	coreNamespaceName := utils.ResolveCoreNamespaceName(t.Name)
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

// This creates a role binding for the tenant. The role binding will be created inside the core namespace of the
// tenant. By this way the tenant's permissions will be contained inside the core namespace.
func (m *multiTenancyManager) CreateTenantAdminRoleBinding(ctx context.Context, t *v1.Tenant) error {
	// Retrieve the role bingind, if already exists, do nothing.
	roleBinding := &rbacv1.RoleBinding{}
	err := m.client.Get(ctx,
		types.NamespacedName{
			// The name of the rolebinding should be TenantAdminRoleName and Namespace should be core namespace
			Name:      v1.TenantAdminRoleName,
			Namespace: utils.ResolveCoreNamespaceName(t.GetName()),
		}, roleBinding)

	// If it doesn't exsist then create it.
	if err != nil {
		// Create the role binding
		if errors.IsNotFound(err) {
			roleBinding = &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      v1.TenantAdminRoleName,
					Namespace: utils.ResolveCoreNamespaceName(t.GetName()),
					Labels: map[string]string{
						"edge-net.io/generated":    "true",
						"edge-net.io/notification": "true",
					},
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:     "User",
						APIGroup: "rbac.authorization.k8s.io",
						Name:     t.Spec.Admin,
					},
				},
				RoleRef: rbacv1.RoleRef{
					Kind: "ClusterRole",
					Name: v1.TenantAdminRoleName,
				},
			}

			return m.client.Create(ctx, roleBinding)
		}
		return err
	}
	return nil
}

// Create the network policy, if specified in the tenant create the cluster network policy as well.
func (m *multiTenancyManager) CreateTenantNetworkPolicy(ctx context.Context, t *v1.Tenant) error {
	clusterUID, err := utils.GetClusterUID(ctx, m.client)
	if err != nil {
		return err
	}

	port := intstr.IntOrString{IntVal: 1}
	endPort := int32(32768)
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"edge-net.io/subtenant":   "false",
			"edge-net.io/tenant":      t.GetName(),
			"edge-net.io/tenant-uid":  string(t.GetUID()),
			"edge-net.io/cluster-uid": string(clusterUID),
		},
	}

	// Create a new network policy object.
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			// The name is fixed to baseline
			Name: "baseline",
			// Create the policy in the tenant's core namespace
			Namespace: utils.ResolveCoreNamespaceName(t.GetName()),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{"Ingress"},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &labelSelector,
						},
						{
							IPBlock: &networkingv1.IPBlock{
								CIDR:   "0.0.0.0/0",
								Except: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port:    &port,
							EndPort: &endPort,
						},
					},
				},
			},
		},
	}

	// Only return if there is an error other than already exists
	if err = m.client.Create(ctx, networkPolicy); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	ownerReferences := []metav1.OwnerReference{
		*metav1.NewControllerRef(t, t.GroupVersionKind()),
	}
	dropAction := antreav1alpha1.RuleActionDrop
	allowAction := antreav1alpha1.RuleActionAllow

	clusterNetworkPolicy := antreav1alpha1.ClusterNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "baseline",
			OwnerReferences: ownerReferences,
		},
		Spec: antreav1alpha1.ClusterNetworkPolicySpec{
			Tier:     "tenant",
			Priority: 5,
			AppliedTo: []antreav1alpha1.AppliedTo{
				{
					NamespaceSelector: &labelSelector,
				},
			},
			Ingress: []antreav1alpha1.Rule{
				{
					Action: &allowAction,
					From: []antreav1alpha1.NetworkPolicyPeer{
						{
							NamespaceSelector: &labelSelector,
						},
					},
					Ports: []antreav1alpha1.NetworkPolicyPort{
						{
							Port:    &port,
							EndPort: &endPort,
						},
					},
				},
				{
					Action: &dropAction,
					From: []antreav1alpha1.NetworkPolicyPeer{
						{
							IPBlock: &antreav1alpha1.IPBlock{
								CIDR: "10.0.0.0/8",
							},
						},
						{
							IPBlock: &antreav1alpha1.IPBlock{
								CIDR: "172.16.0.0/12",
							},
						},
						{
							IPBlock: &antreav1alpha1.IPBlock{
								CIDR: "192.168.0.0/16",
							},
						},
					},
					Ports: []antreav1alpha1.NetworkPolicyPort{
						{
							Port:    &port,
							EndPort: &endPort,
						},
					},
				},
				{
					Action: &allowAction,
					From: []antreav1alpha1.NetworkPolicyPeer{
						{
							IPBlock: &antreav1alpha1.IPBlock{
								CIDR: "0.0.0.0/0",
							},
						},
					},
					Ports: []antreav1alpha1.NetworkPolicyPort{
						{
							Port:    &port,
							EndPort: &endPort,
						},
					},
				},
			},
		},
	}

	// Check if in the tenant spec the cluster network policy is requested. If this is false, try to delete the policy if it exist.
	if t.Spec.ClusterNetworkPolicy {
		if err = m.client.Create(ctx, &clusterNetworkPolicy); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	} else {
		if err = m.client.Delete(ctx, &clusterNetworkPolicy); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
