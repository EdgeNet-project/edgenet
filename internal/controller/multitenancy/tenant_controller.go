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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	multitenancyv1 "github.com/edgenet-project/edgenet-software/api/multitenancy/v1"
	"github.com/edgenet-project/edgenet-software/internal/multitenancy/v1"
	"github.com/edgenet-project/edgenet-software/internal/utils"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// These are required to have the permissions.
//+kubebuilder:rbac:groups=multitenancy.edge-net.io,resources=tenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies;clusternetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="crd.antrea.io",resources=clusternetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multitenancy.edge-net.io,resources=tenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multitenancy.edge-net.io,resources=tenants/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Tenant object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	tenant := multitenancyv1.Tenant{}
	isMarkedForDeletion, reconcileResult, err := utils.GetResourceWithFinalizer(ctx, r.Client, &tenant, req.NamespacedName)

	if !utils.IsObjectInitialized(&tenant) {
		return reconcileResult, err
	}

	multiTenancyManager, err := multitenancy.NewMultiTenancyManager(ctx, r.Client)

	if err != nil {
		return ctrl.Result{}, err
	}

	if isMarkedForDeletion {
		// Do a cleanup and allow tenant object for deletion
		if err := multiTenancyManager.TenantCleanup(ctx, &tenant); err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		return utils.AllowObjectDeletion(ctx, r.Client, &tenant)
	} else {
		// Create a core namespace for the tenant.
		if err := multiTenancyManager.CreateCoreNamespaceLocal(ctx, &tenant); err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		// Create the role binding in the core namespace of the tenant.
		if err := multiTenancyManager.CreateTenantAdminRoleBinding(ctx, &tenant); err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		// Create the network policy. This restricts pod communication. Don't need to clean after
		// deletion of the tenant.
		if err := multiTenancyManager.CreateTenantNetworkPolicy(ctx, &tenant); err != nil {
			return ctrl.Result{Requeue: true}, err
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenancyv1.Tenant{}).
		Complete(r)
}
