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

package labeller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This interface contains the necessary functions to perform the operations related to
// labelling. Most of the implementation here is retrieved from the old implementation.
// However, some of the functions are changed.
type LabelManager interface {
	// This adds the labes to the node object given
	LabelNode(context.Context, *corev1.Node) error
}

type labelManager struct {
	LabelManager
	client client.Client
}

func NewLabelManager(ctx context.Context, client client.Client) (LabelManager, error) {
	return &labelManager{
		client: client,
	}, nil
}

// This adds the labels to the node and updates it. If any error occures it returnes the error.
func (m *labelManager) LabelNode(context.Context, *corev1.Node) error {
	return nil
}
