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

package deployments

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	deploymentsv1 "github.com/edgenet-project/edgenet-software/api/deployments/v1"
	"github.com/edgenet-project/edgenet-software/internal/deployments/v1"
	"github.com/edgenet-project/edgenet-software/internal/utils"
)

// SelectiveDeploymentReconciler reconciles a SelectiveDeployment object
type SelectiveDeploymentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=deployments.edge-net.io,resources=selectivedeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=deployments.edge-net.io,resources=selectivedeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=deployments.edge-net.io,resources=selectivedeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SelectiveDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *SelectiveDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	sd := deploymentsv1.SelectiveDeployment{}
	isMarkedForDeletion, reconcileResult, err := utils.GetResourceWithFinalizer(ctx, r.Client, &sd, req.NamespacedName)

	if !utils.IsObjectInitialized(&sd) {
		return reconcileResult, err
	}

	deploymentManager, err := deployments.NewDeploymentManager(ctx, r.Client)

	if err != nil {
		logger.Error(err, "cannot create deployment manager")
		return ctrl.Result{}, nil
	}

	if isMarkedForDeletion {
		if err := deploymentManager.SelectiveDeploymentCleanup(ctx, &sd); err != nil {
			utils.RecordEventError(r.recorder, &sd, "Selective Deployment cleanup failed")
			return ctrl.Result{Requeue: true}, err
		}

		return utils.AllowObjectDeletion(ctx, r.Client, &sd)
	} else {
		logger.Info("Reconciliation...")
		// See how SD is implemented...
		// TODO
	}

	utils.RecordEventInfo(r.recorder, &sd, "Selective Deployment reconciliation successfull")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SelectiveDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Setup the event recorder
	r.recorder = utils.GetEventRecorder(mgr)
	return ctrl.NewControllerManagedBy(mgr).
		For(&deploymentsv1.SelectiveDeployment{}).
		Complete(r)
}
