/*
Copyright 2024 EdgeNet.

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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	multitenancyedgenetiov1 "github.com/ubombar/edgenet-kubebuilder/api/v1"
	v1 "github.com/ubombar/edgenet-kubebuilder/api/v1"
	"github.com/ubombar/edgenet-kubebuilder/internal/multitenancy"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=multitenancy.edge-net.io.edge-net.io,resources=tenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multitenancy.edge-net.io.edge-net.io,resources=tenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multitenancy.edge-net.io.edge-net.io,resources=tenants/finalizers,verbs=update

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
	_ = log.FromContext(ctx)

	multitenancyManager, err := multitenancy.NewMultiTenancyManager(ctx, r.Client)

	if err != nil {
		return reconcile.Result{}, nil
	}

	// Initialize the empty tenant object
	tenant := v1.Tenant{}

	// If the resource cannot be found then it means it is either deleted or there is a io error.
	// Requeue if it is not a not found error.
	if err := r.Get(ctx, req.NamespacedName, &tenant); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{
			Requeue: false,
		}, err
	}

	// // Check if the object is marked for deletion
	// if !tenant.DeletionTimestamp.IsZero() {
	// }

	err = multitenancyManager.CreateCoreNamespace(ctx, &tenant)
	fmt.Printf("err: %v\n", err)

	fmt.Printf("successfully retrieved the tenant %q\n", tenant.Spec.FullName)

	return ctrl.Result{
		Requeue: false,
	}, nil
}

func (r *TenantReconciler) OnDeletion(t *v1.Tenant) (ctrl.Result, error) {
	return reconcile.Result{}, nil
}

func (r *TenantReconciler) OnUpdate(t *v1.Tenant) (ctrl.Result, error) {
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenancyedgenetiov1.Tenant{}).
		Complete(r)
}
