package utils

import (
	"context"

	v1 "github.com/ubombar/edgenet-kubebuilder/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func ContainsFinalizer(finalizers []string, finalizer string) bool {
	for _, item := range finalizers {
		if item == finalizer {
			return true
		}
	}
	return false
}

func RemoveFinalizer(finalizers []string, finalizer string) []string {
	for i, item := range finalizers {
		if item == finalizer {
			// Remove the item at index i from slice.
			return append(finalizers[:i], finalizers[i+1:]...)
		}
	}
	// Return the original slice if the string is not found.
	return finalizers
}

// func GetResourceWithFinalizer(ctx context.Context, c client.Client, namespacedName types.NamespacedName) (*v1.Tenant, bool, ctrl.Result, error) {
// 	objOld := &v1.Tenant{}
// 	if err := c.Get(ctx, namespacedName, objOld); err != nil {
// 		if errors.IsNotFound(err) {
// 			return nil, false, reconcile.Result{}, nil
// 		}
// 		return nil, false, reconcile.Result{Requeue: true}, err
// 	}

// 	obj := objOld.DeepCopy()

// 	// If the object is not marked for deletion
// 	if obj.DeletionTimestamp.IsZero() && !ContainsFinalizer(obj.ObjectMeta.Finalizers, "edge-net.io/controller") {
// 		obj.ObjectMeta.Finalizers = append(obj.ObjectMeta.Finalizers, "edge-net.io/controller")

// 		if err := c.Update(ctx, obj); err != nil {
// 			return nil, false, reconcile.Result{Requeue: true}, err
// 		}
// 	}

// 	return obj, !obj.DeletionTimestamp.IsZero(), reconcile.Result{Requeue: false}, nil
// }

func GetResourceWithFinalizer(ctx context.Context, c client.Client, namespacedName types.NamespacedName) (*v1.Tenant, bool, ctrl.Result, error) {
	obj := &v1.Tenant{}

	if err := c.Get(ctx, namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return nil, false, reconcile.Result{}, nil
		}
		return nil, false, reconcile.Result{Requeue: true}, err
	}

	objCopy := obj.DeepCopy()

	// If the object is not marked for deletion
	if objCopy.GetDeletionTimestamp().IsZero() && !ContainsFinalizer(objCopy.GetFinalizers(), "edge-net.io/controller") {
		objCopy.SetFinalizers(append(objCopy.GetFinalizers(), "edge-net.io/controller"))

		if err := c.Update(ctx, objCopy); err != nil {
			return nil, false, reconcile.Result{Requeue: true}, err
		}
	}

	return objCopy, !objCopy.GetDeletionTimestamp().IsZero(), reconcile.Result{Requeue: false}, nil
}

func ReleaseResource(ctx context.Context, c client.Client, obj *v1.Tenant) (reconcile.Result, error) {
	objCopy := obj.DeepCopy()

	objCopy.ObjectMeta.Finalizers = RemoveFinalizer(objCopy.ObjectMeta.Finalizers, "edge-net.io/controller")

	if err := c.Update(ctx, objCopy.DeepCopy()); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// TODO Try to nmake this generic
// func GetResourceWithFinalizer[T client.Object](ctx context.Context, c client.Client, namespacedName types.NamespacedName) (T, bool, ctrl.Result, error) {
// 	obj := *new(T)

// 	if err := c.Get(ctx, namespacedName, obj); err != nil {
// 		if errors.IsNotFound(err) {
// 			return nil, false, reconcile.Result{}, nil
// 		}
// 		return nil, false, reconcile.Result{Requeue: true}, err
// 	}

// 	// If the object is not marked for deletion
// 	if obj.GetDeletionTimestamp().IsZero() && !ContainsFinalizer(obj.GetFinalizers(), "edge-net.io/controller") {
// 		obj.SetFinalizers(append(obj.GetFinalizers(), "edge-net.io/controller"))

// 		if err := c.Update(ctx, obj); err != nil {
// 			return nil, false, reconcile.Result{Requeue: true}, err
// 		}
// 	}

// 	return obj, !obj.GetDeletionTimestamp().IsZero(), reconcile.Result{Requeue: false}, nil
// }

func GenericGetResourceWithFinalizer(ctx context.Context, c client.Client, obj client.Object, namespacedName types.NamespacedName) (bool, ctrl.Result, error) {
	if err := c.Get(ctx, namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return false, reconcile.Result{}, nil
		}
		return false, reconcile.Result{Requeue: true}, err
	}

	// objCopy := obj.DeepCopyObject()

	// If the object is not marked for deletion
	if obj.GetDeletionTimestamp().IsZero() && !ContainsFinalizer(obj.GetFinalizers(), "edge-net.io/controller") {
		obj.SetFinalizers(append(obj.GetFinalizers(), "edge-net.io/controller"))

		if err := c.Update(ctx, obj); err != nil {
			return false, reconcile.Result{Requeue: true}, err
		}
	}

	return !obj.GetDeletionTimestamp().IsZero(), reconcile.Result{Requeue: false}, nil
}

func GenericReleaseResource(ctx context.Context, c client.Client, obj client.Object) (reconcile.Result, error) {
	obj.SetFinalizers(RemoveFinalizer(obj.GetFinalizers(), "edge-net.io/controller"))

	if err := c.Update(ctx, obj); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
