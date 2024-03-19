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

package labellers

import (
	"context"

	"github.com/edgenet-project/edgenet-software/internal/labeller"
	"github.com/edgenet-project/edgenet-software/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
)

// NodeLabellerReconciler reconciles a NodeLabeller object
type NodeLabellerReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	MaxMind labeller.MaxMind
}

//+kubebuilder:rbac:groups=core,resources=node,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeLabeller object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *NodeLabellerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Get the node directly
	node := &corev1.Node{}

	if err := utils.GetResource(ctx, r.Client, node, req.NamespacedName); err != nil {
		return ctrl.Result{}, err
	}

	// Create the labeller manager
	labellerManager, err := labeller.NewLabelManager(ctx, r.Client)

	if err != nil {
		return ctrl.Result{}, err
	}

	// Label the node
	if err := labellerManager.LabelNode(ctx, node); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeLabellerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&corev1.Node{}).
		Complete(r)
}
