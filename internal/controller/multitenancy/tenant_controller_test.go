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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	multitenancyv1 "github.com/edgenet-project/edgenet-software/api/multitenancy/v1"
)

var _ = Describe("Tenant Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		tenant := &multitenancyv1.Tenant{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Tenant")
			err := k8sClient.Get(ctx, typeNamespacedName, tenant)
			if err != nil && errors.IsNotFound(err) {
				resource := &multitenancyv1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: multitenancyv1.TenantSpec{
						FullName:             "Test User",
						Description:          "This is the description of the test user.",
						Admin:                "testuser",
						URL:                  "https://example.com",
						Enabled:              true,
						InitialRequest:       map[v1.ResourceName]resource.Quantity{},
						ClusterNetworkPolicy: false,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			// resource := &multitenancyv1.Tenant{}
			// err := k8sClient.Get(ctx, typeNamespacedName, resource)
			// Expect(err).NotTo(HaveOccurred())

			// By("Cleanup the specific resource instance Tenant")
			// Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			// By("Reconciling the created resource")
			// controllerReconciler := &TenantReconciler{
			// 	Client: k8sClient,
			// 	Scheme: k8sClient.Scheme(),
			// }

			// _, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
			// 	NamespacedName: typeNamespacedName,
			// })
			// Expect(err).NotTo(HaveOccurred())
			// // TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// // Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
