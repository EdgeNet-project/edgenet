package multitenancy

import "fmt"

// Resolve the core-namespace from tenant name (simply take the object name and add prefix "core-")
func ResolveCoreNamespaceName(tenantName string) string {
	return fmt.Sprintf("core-%s", tenantName)
}
