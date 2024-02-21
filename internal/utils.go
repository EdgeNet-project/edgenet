package utils

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func containsFinalizer(finalizers []string, finalizer string) bool {
	for _, item := range finalizers {
		if item == finalizer {
			return true
		}
	}
	return false
}

func removeFinalizer(finalizers []string, finalizer string) []string {
	for i, item := range finalizers {
		if item == finalizer {
			// Remove the item at index i from slice.
			return append(finalizers[:i], finalizers[i+1:]...)
		}
	}
	// Return the original slice if the string is not found.
	return finalizers
}

// Gets the requested object, adds a finalizer if not already present.
func GetResourceWithFinalizer(ctx context.Context, c client.Client, obj client.Object, namespacedName types.NamespacedName) (bool, ctrl.Result, error) {
	if err := c.Get(ctx, namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return false, reconcile.Result{}, nil
		}
		return false, reconcile.Result{Requeue: true}, err
	}

	// If the object is not marked for deletion
	if obj.GetDeletionTimestamp().IsZero() && !containsFinalizer(obj.GetFinalizers(), "edge-net.io/controller") {
		obj.SetFinalizers(append(obj.GetFinalizers(), "edge-net.io/controller"))

		if err := c.Update(ctx, obj); err != nil {
			return false, reconcile.Result{Requeue: true}, err
		}
	}

	return !obj.GetDeletionTimestamp().IsZero(), reconcile.Result{Requeue: false}, nil
}

// Removes the finalizer from the object.
func RemoveFinalizer(ctx context.Context, c client.Client, obj client.Object) (reconcile.Result, error) {
	obj.SetFinalizers(removeFinalizer(obj.GetFinalizers(), "edge-net.io/controller"))

	if err := c.Update(ctx, obj); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
