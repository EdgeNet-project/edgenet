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

	deploymentsv1 "github.com/edgenet-project/edgenet-software/api/deployments/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This interface contains the necessary functions to perform the operations related to
// deployments. Most of the implementation here is retrieved from the old implementation.
// However, some of the functions are changed.
type DeploymentManager interface {
	// Cleanup the Selective Deployment's derivatives.
	SelectiveDeploymentCleanup(context.Context, *deploymentsv1.SelectiveDeployment) error
}

type deploymentManager struct {
	DeploymentManager
	client client.Client
}

func NewDeploymentManager(ctx context.Context, client client.Client) (DeploymentManager, error) {
	return &deploymentManager{
		client: client,
	}, nil
}

func (m *deploymentManager) SelectiveDeploymentCleanup(ctx context.Context, sd *deploymentsv1.SelectiveDeployment) error {
	return nil
}
