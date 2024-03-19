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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This interface contains the necessary functions to perform the operations related to
// labelling. Most of the implementation here is retrieved from the old implementation.
// However, some of the functions are changed.
type LabelManager interface {
	// This adds the labes to the node object given
	LabelNode(context.Context, *corev1.Node) error

	// Gets the internal and external ip address
	GetNodeIPAddresses(obj *corev1.Node) (string, string)
}

type labelManager struct {
	LabelManager
	client  client.Client
	Maxmind MaxMind
}

func NewLabelManager(ctx context.Context, client client.Client, maxmind MaxMind) (LabelManager, error) {
	return &labelManager{
		client:  client,
		Maxmind: maxmind,
	}, nil
}

// This adds the labels to the node and updates it. If any error occures it returnes the error.
func (m *labelManager) LabelNode(ctx context.Context, node *corev1.Node) error {
	fmt.Println("Node labeller implementation required...")

	// TODO: Get this part of the code from old repo
	// internalIp, externalIp := m.GetNodeIPAddresses(node)

	// Old code...
	// // 1. Use the VPNPeer endpoint address if available.
	// peer, err := c.edgenetclientset.NetworkingV1alpha1().VPNPeers().Get(context.TODO(), nodeObj.Name, v1.GetOptions{})
	// if err != nil {
	// 	klog.V(4).Infof(
	// 		"Failed to find a matching VPNPeer object for %s: %s. The node IP will be used instead.",
	// 		nodeObj.Name,
	// 		err,
	// 	)
	// } else {
	// 	klog.V(4).Infof("VPNPeer endpoint IP: %s", *peer.Spec.EndpointAddress)
	// 	result = multiproviderManager.GetGeolocationByIP(
	// 		c.maxmindURL,
	// 		c.maxmindAccountID,
	// 		c.maxmindLicenseKey,
	// 		nodeObj.Name,
	// 		*peer.Spec.EndpointAddress,
	// 	)
	// }

	// // 2. Otherwise use the node external IP if available.
	// if externalIP != "" && !result {
	// 	klog.V(4).Infof("External IP: %s", externalIP)
	// 	result = multiproviderManager.GetGeolocationByIP(
	// 		c.maxmindURL,
	// 		c.maxmindAccountID,
	// 		c.maxmindLicenseKey,
	// 		nodeObj.Name,
	// 		externalIP,
	// 	)
	// }

	// // 3. Otherwise use the node internal IP if available.
	// if internalIP != "" && !result {
	// 	klog.V(4).Infof("Internal IP: %s", internalIP)
	// 	multiproviderManager.GetGeolocationByIP(
	// 		c.maxmindURL,
	// 		c.maxmindAccountID,
	// 		c.maxmindLicenseKey,
	// 		nodeObj.Name,
	// 		internalIP,
	// 	)
	// }

	return nil
}

// GetNodeIPAddresses picks up the internal and external IP addresses of the Node
func (m *labelManager) GetNodeIPAddresses(obj *corev1.Node) (string, string) {
	internalIP := ""
	externalIP := ""
	for _, addressesRow := range obj.Status.Addresses {
		if addressType := addressesRow.Type; addressType == "InternalIP" {
			internalIP = addressesRow.Address
		}
		if addressType := addressesRow.Type; addressType == "ExternalIP" {
			externalIP = addressesRow.Address
		}
	}
	return internalIP, externalIP
}
