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

package utils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Define a custom type that implements the flag.Value interface
type FlagList []string

// Implement the Set method for the flag.Value interface
func (s *FlagList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Implement the String method for the flag.Value interface
func (s *FlagList) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *FlagList) Contains(flag string) bool {
	for _, f := range *s {
		if f == flag {
			return true
		}
	}
	return false
}

// Resolve the core-namespace from tenant name (simply take the object name)
func ResolveCoreNamespaceName(tenantName string) string {
	return tenantName
}

// Check a string exists in a list of strings
func containsFinalizer(finalizers []string, finalizer string) bool {
	for _, item := range finalizers {
		if item == finalizer {
			return true
		}
	}
	return false
}

// Remove a string from a list of strings
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

// Gets the object from by using the given client c and the address of the obj as well as namespacedName variable.
// Populates the obj variable if the object exists.
// Return values are (isDeleted, response, error)
// Intended use:
//
//	tenant := &v1.Tenant{}
//	isMarkedForDeletion, res, err := utils.GetResourceWithFinalizer(
//		ctx,
//		client,
//		tenant,
//		namespacedName)
//
//	if !utils.IsObjectInitialized(tenant) {
//		return res, err
//	}
func GetResourceWithFinalizer(ctx context.Context, c client.Client, obj client.Object, namespacedName types.NamespacedName) (bool, ctrl.Result, error) {
	// Get the object from the cluster
	if err := c.Get(ctx, namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return false, reconcile.Result{}, nil
		}
		return false, reconcile.Result{Requeue: true}, err
	}

	// If the object is not marked for deletion and doesn't contain the finalizer
	if obj.GetDeletionTimestamp().IsZero() && !containsFinalizer(obj.GetFinalizers(), "edge-net.io/controller") {
		obj.SetFinalizers(append(obj.GetFinalizers(), "edge-net.io/controller"))

		if err := c.Update(ctx, obj); err != nil {
			return false, reconcile.Result{Requeue: true}, err
		}
	}

	// First return represent if the object is marked for deletion,
	// Second is the reconsitiation result without requeue.
	// Third is the error which is nil in this case.
	return !obj.GetDeletionTimestamp().IsZero(), reconcile.Result{Requeue: false}, nil
}

// Normally when a Kubernetes object is deleted it is no longer accessible from the etcd. To retrieve the last state
// of the object finalizers are used. GetResourceWithFinalizer function adds a finalizer to the resource if not present.
// These finalizers are then can used to keep the object in the cluster after it is marked for deletion.
// AllowObjectDeletion method removes the edge-net.io/controller finalizer. By this way the object can be completely removed
// from the cluster.
func AllowObjectDeletion(ctx context.Context, c client.Client, obj client.Object) (reconcile.Result, error) {
	obj.SetFinalizers(removeFinalizer(obj.GetFinalizers(), "edge-net.io/controller"))

	if err := c.Update(ctx, obj); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// Checks if the given object is an initializer kubernetes object. This is done by checking the UID of the object.
// If it is an empty string (default value for UID) then it is not initialized and returns false. Otherwise true.
func IsObjectInitialized(obj client.Object) bool {
	return obj.GetUID() != ""
}

// Gets the UID of the kube-system namesapce. This namespace is considered a unique identifier of the cluster.
func GetClusterUID(ctx context.Context, client client.Client) (types.UID, error) {
	namespace := corev1.Namespace{}
	err := client.Get(ctx, types.NamespacedName{Name: "kube-system"}, &namespace)
	if err != nil {
		return types.UID(""), err
	}

	return namespace.GetUID(), nil
}

// Gets the event recorder from the manager. This event recorder is used for sending events for objects.
func GetEventRecorder(mgr ctrl.Manager) record.EventRecorder {
	return mgr.GetEventRecorderFor("edgenet-controller")
}

// Sends an error to the object using the event recorder. The type is error.
func RecordEventError(r record.EventRecorder, obj client.Object, message string) {
	r.Eventf(obj, "Warning", "Error Occured", message)
}

// Sends an update event to the object using the event recorder.
func RecordEventInfo(r record.EventRecorder, obj client.Object, message string) {
	r.Eventf(obj, "Normal", "Update Occured", message)
}
