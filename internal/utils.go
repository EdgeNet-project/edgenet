package util

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
