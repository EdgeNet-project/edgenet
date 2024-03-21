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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	multitenancyv1 "github.com/edgenet-project/edgenet/api/multitenancy/v1"
	"github.com/edgenet-project/edgenet/internal/multitenancy/v1"
	"github.com/edgenet-project/edgenet/internal/utils"
)

// SubNamespaceReconciler reconciles a SubNamespace object
type SubNamespaceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=multitenancy.edge-net.io,resources=subnamespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multitenancy.edge-net.io,resources=subnamespaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multitenancy.edge-net.io,resources=subnamespaces/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SubNamespace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *SubNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	sns := multitenancyv1.SubNamespace{}
	isMarkedForDeletion, reconcileResult, err := utils.GetResourceWithFinalizer(ctx, r.Client, &sns, req.NamespacedName)

	if !utils.IsObjectInitialized(&sns) {
		return reconcileResult, err
	}

	multiTenancyManager, err := multitenancy.NewMultiTenancyManager(ctx, r.Client)

	if err != nil {
		logger.Error(err, "cannot create multitenancy manager")
		return ctrl.Result{}, err
	}

	if isMarkedForDeletion {
		// Do a cleanup and allow tenant object for deletion
		if err := multiTenancyManager.SubNamespaceCleanup(ctx, &sns); err != nil {
			utils.RecordEventError(r.recorder, &sns, "SubNamespace cleanup failed")
			return ctrl.Result{Requeue: true}, err
		}

		return utils.AllowObjectDeletion(ctx, r.Client, &sns)
	} else {
		// TODO: What to do now?
		if err := multiTenancyManager.SetupSubNamespace(ctx, &sns); err != nil {
			utils.RecordEventError(r.recorder, &sns, "SubNamespace setup failed")
			return ctrl.Result{Requeue: true}, err
		}
	}

	utils.RecordEventInfo(r.recorder, &sns, "SubNamespace reconciliation successfull")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Setup the event recorder
	r.recorder = utils.GetEventRecorder(mgr)

	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenancyv1.SubNamespace{}).
		Complete(r)
}
