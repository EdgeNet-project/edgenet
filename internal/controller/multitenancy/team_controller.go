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

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// These are required to have the permissions.
//+kubebuilder:rbac:groups="multitenancy.edge-net.io",resources=teams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies;clusternetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="crd.antrea.io",resources=clusternetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="multitenancy.edge-net.io",resources=teams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="multitenancy.edge-net.io",resources=teams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Team object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	team := multitenancyv1.Team{}
	isMarkedForDeletion, reconcileResult, err := utils.GetResourceWithFinalizer(ctx, r.Client, &team, req.NamespacedName)

	if !utils.IsObjectInitialized(&team) {
		return reconcileResult, err
	}

	multiTenancyManager, err := multitenancy.NewMultiTenancyManager(ctx, r.Client)

	if err != nil {
		l.Error(err, "cannot create multitenancy manager")
		return ctrl.Result{}, err
	}

	if isMarkedForDeletion {
		// Do a cleanup and allow team object for deletion
		if err := multiTenancyManager.TeamCleanup(ctx, &team); err != nil {
			utils.RecordEventError(&l, r.recorder, &team, "Team cleanup failed")
			return ctrl.Result{Requeue: true}, err
		}

		return utils.AllowObjectDeletion(ctx, r.Client, &team)
	} else {
		// Create a core namespace for the team.
		if err := multiTenancyManager.CreateCoreNamespaceLocal(ctx, &team); err != nil {
			utils.RecordEventError(&l, r.recorder, &team, "Team Core Namespace creation failed")
			return ctrl.Result{Requeue: true}, err
		}

		// Create the role binding in the core namespace of the team.
		if err := multiTenancyManager.CreateTeamAdminRoleBinding(ctx, &team); err != nil {
			utils.RecordEventError(&l, r.recorder, &team, "Team admin role binding failed")
			return ctrl.Result{Requeue: true}, err
		}

		// Create the network policy. This restricts pod communication. Don't need to clean after
		// deletion of the team.
		if err := multiTenancyManager.CreateTeamNetworkPolicy(ctx, &team); err != nil {
			utils.RecordEventError(&l, r.recorder, &team, "Team antrea network policy failed")
			return ctrl.Result{Requeue: true}, err
		}
	}

	utils.RecordEventInfo(&l, r.recorder, &team, "Team reconciliation successfull")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Setup the event recorder
	r.recorder = utils.GetEventRecorder(mgr)

	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenancyv1.Team{}).
		Complete(r)
}
