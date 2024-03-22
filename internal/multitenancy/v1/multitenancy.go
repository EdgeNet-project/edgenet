/*
Copyright 2024 Contributors to EdgeNet Project.

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

package multitenancy

import (
	"context"
	errors2 "errors"

	antreav1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	multitenancyv1 "github.com/edgenet-project/edgenet/api/multitenancy/v1"
	"github.com/edgenet-project/edgenet/internal/utils"
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
	TenantCleanup(context.Context, *multitenancyv1.Tenant) error

	// Creates a core namespace (same name with the tenant) and sets the resource allocation.
	// Returns nil, if the namespace already exists.
	CreateCoreNamespace(context.Context, *multitenancyv1.Tenant, types.UID) error

	// Same as the CreateCoreNamespace except gets the UID from local cluster.
	CreateCoreNamespaceLocal(context.Context, *multitenancyv1.Tenant) error

	// Creates a new tenant role binding with admin priviliages. Requires "edgenet:tenant-admin" role
	// to work.
	CreateTenantAdminRoleBinding(context.Context, *multitenancyv1.Tenant) error

	// Create the network policy. If specified creates the cluster network policy as well.
	CreateTenantNetworkPolicy(context.Context, *multitenancyv1.Tenant) error

	// Cleanups the SubNamespace
	SubNamespaceCleanup(context.Context, *multitenancyv1.SubNamespace) error

	// Creates and setups studd for the SubNamespace
	SetupSubNamespace(context.Context, *multitenancyv1.SubNamespace) error
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

func (m *multiTenancyManager) TenantCleanup(ctx context.Context, t *multitenancyv1.Tenant) error {
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
func (m *multiTenancyManager) CreateCoreNamespaceLocal(ctx context.Context, t *multitenancyv1.Tenant) error {
	clusterUID, err := utils.GetClusterUID(ctx, m.client)

	if err != nil {
		return err
	}

	return m.CreateCoreNamespace(ctx, t, clusterUID)
}

// Creates a core namespace, sets the ownership references and does resource allocation.
// The clusterUID is given as a future federation concept. Also creates a ResourceQuota object.
func (m *multiTenancyManager) CreateCoreNamespace(ctx context.Context, t *multitenancyv1.Tenant, clusterUID types.UID) error {
	// Get the corenamespace name
	coreNamespaceName := utils.ResolveCoreNamespaceName(t.Name)
	coreNamespaceObjectKey := client.ObjectKey{Name: coreNamespaceName}

	// Create the namespace
	coreNamespace := corev1.Namespace{}
	err := m.client.Get(ctx, coreNamespaceObjectKey, &coreNamespace)

	labels := map[string]string{
		"edge-net.io/tenant":      t.GetName(),
		"edge-net.io/kind":        "core",
		"edge-net.io/tenant-uid":  string(t.GetUID()),
		"edge-net.io/cluster-uid": string(clusterUID),
		"edge-net.io/generated":   "true",
	}

	// If there is no core namespace, create one
	if err != nil && errors.IsNotFound(err) {
		// Create the namespace here
		coreNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: coreNamespaceName,
				// Set the labels
				Labels: labels,
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

	// Set the resource quota for the namespace. Note that the resource quota is not additive in the namespace.
	// Kubernetes takes the smallest one to check if a pod within bounds. Therefore we might want to use a custom
	// object like SmartResourceQuota.
	rc := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreNamespaceName,
			Namespace: coreNamespaceName,
			Labels:    labels,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: t.Spec.InitialRequest,
		},
	}

	// Try to create the resource quota specified in the initial request. If it already exists, try to change it.
	// if there is still and error return the error.
	if err := m.client.Create(ctx, rc); err != nil {
		if errors.IsAlreadyExists(err) {
			if err := m.client.Update(ctx, rc); err != nil {
				return err
			}
		} else {
			return err
		}

	}

	return nil
}

// This creates a role binding for the tenant. The role binding will be created inside the core namespace of the
// tenant. By this way the tenant's permissions will be contained inside the core namespace.
func (m *multiTenancyManager) CreateTenantAdminRoleBinding(ctx context.Context, t *multitenancyv1.Tenant) error {
	// Retrieve the role bingind, if already exists, do nothing.
	roleBinding := &rbacv1.RoleBinding{}
	err := m.client.Get(ctx,
		types.NamespacedName{
			// The name of the rolebinding should be TenantAdminRoleName and Namespace should be core namespace
			Name:      multitenancyv1.TenantAdminRoleName,
			Namespace: utils.ResolveCoreNamespaceName(t.GetName()),
		}, roleBinding)

	// If it doesn't exsist then create it.
	if err != nil {
		// Create the role binding
		if errors.IsNotFound(err) {
			roleBinding = &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      multitenancyv1.TenantAdminRoleName,
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
					Name: multitenancyv1.TenantAdminRoleName,
				},
			}

			return m.client.Create(ctx, roleBinding)
		}
		return err
	}
	return nil
}

// Create the network policy, if specified in the tenant create the cluster network policy as well.
func (m *multiTenancyManager) CreateTenantNetworkPolicy(ctx context.Context, t *multitenancyv1.Tenant) error {
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

// Deletes the created child namespace.
func (m *multiTenancyManager) SubNamespaceCleanup(ctx context.Context, s *multitenancyv1.SubNamespace) error {
	subNamespaceName := utils.ResolveSubNamespaceName(s)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: subNamespaceName,
		},
	}

	// Then finally try to delete the namespace
	if err := m.client.Delete(ctx, ns); err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

// This creates a new namespace using the generated name. Then populates the namespace with the initial allocation.
// Then gives the current tenant admin the permissions.
func (m *multiTenancyManager) SetupSubNamespace(ctx context.Context, s *multitenancyv1.SubNamespace) error {
	subNamespaceName := utils.ResolveSubNamespaceName(s)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: subNamespaceName,
			Labels: map[string]string{
				"edge-net.io/generated": "true",
				"edge-net.io/kind":      "sub",
				"edge-net.io/parent":    s.GetNamespace(),
			},
			// We will use admission controller to prevent namespaces that are managed by the subnamespace controller from deleting.
			// So we will not have finalizers and owners in the newly created object.
		},
	}

	// Try to create the namespace, continue even if it already exist
	if err := m.client.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// TODO: Create the role bingind etc.
	t, err := m.getRootTenant(ctx, s)

	if err != nil {
		return err
	}

	// Retrieve the role bingind, if already exists, do nothing.
	roleBinding := &rbacv1.RoleBinding{}
	err = m.client.Get(ctx,
		types.NamespacedName{
			// The name of the rolebinding should be TenantAdminRoleName and Namespace should be core namespace
			Name:      multitenancyv1.TenantAdminRoleName,
			Namespace: subNamespaceName,
		}, roleBinding)

	// If it doesn't exsist then create it.
	if err != nil {
		// Create the role binding
		if errors.IsNotFound(err) {
			roleBinding = &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      multitenancyv1.TenantAdminRoleName,
					Namespace: subNamespaceName,
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
					Name: multitenancyv1.TenantAdminRoleName,
				},
			}

			return m.client.Create(ctx, roleBinding)
		}
		return err
	}
	return nil
}

// Gets the tenant of the topmost namespace in the subnamespace hierarchy.
func (m *multiTenancyManager) getRootTenant(ctx context.Context, s *multitenancyv1.SubNamespace) (*multitenancyv1.Tenant, error) {
	// Start with the current namespace then go up.
	currentNamespaceName := utils.ResolveSubNamespaceName(s)

	// This should be limited in case there happens to be a loop. for now limit this to 255. A Map or a Set can be used here
	// to check if the current namespace is traversed before.
	for i := 0; i < 255; i++ {
		currentNamespace := &corev1.Namespace{}
		// Try to get the namespace, if there is an error immidietly return.
		if err := m.client.Get(ctx, types.NamespacedName{Name: currentNamespaceName}, currentNamespace); err != nil {
			return nil, err
		}

		if namespaceType, ok := currentNamespace.GetLabels()["edge-net.io/kind"]; !ok {
			return nil, errors2.New("currently traversed namespace doesn't have required labels")
		} else {
			// If the type of the namespace is core then get the tenant with the same name
			if namespaceType == "core" {
				tenant := &multitenancyv1.Tenant{}

				// Try to get the tenant with the same name as the tenant.
				if err := m.client.Get(ctx, types.NamespacedName{Name: currentNamespace.Name}, tenant); err != nil {
					return nil, err
				}

				// Happy ending
				return tenant, nil
			} else if namespaceType == "sub" {
				if namespaceParent, ok := currentNamespace.GetLabels()["edge-net.io/parent"]; !ok {
					return nil, errors2.New("cannot get label on namespace, 'edge-net.io/parent'")
				} else {
					// Change the current namespace to the current namespace's parent name, essentially go up in the tree.
					currentNamespaceName = namespaceParent
				}
			} else {
				return nil, errors2.New("unknown label type for namespace, 'edge-net.io/kind' can only be 'core' or 'sub'")
			}
		}
	}

	return nil, errors2.New("subnamespace traverse limit exceeded, there might be a loop in the subnamespace hierarchy")
}
