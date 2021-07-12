/*
Copyright 2021 Contributors to the EdgeNet project.

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

package vpnmesh

import (
	"fmt"
	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"
	edgenetscheme "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned/scheme"
	informers "github.com/EdgeNet-project/edgenet/pkg/generated/informers/externalversions/core/v1alpha"
	listers "github.com/EdgeNet-project/edgenet/pkg/generated/listers/core/v1alpha"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"net"
	"time"
)

const controllerAgentName = "vpnmesh-controller"

const (
	PeerSynced        = "Synced"
	MessagePeerSynced = "VPN peer synced successfully"
)

// Controller is the controller implementation for the VPN mesh.
// This _agent_ runs on every node of the cluster and watches for changes in nodecontribution objects.
// When a nodecontribution object changes, it reconfigures the nodes's VPN interface to peer with the other nodes.
type Controller struct {
	kubeclientset           kubernetes.Interface
	edgenetclientset        clientset.Interface
	nodecontributionsLister listers.NodeContributionLister
	nodecontributionsSynced cache.InformerSynced
	linkname                string
	workqueue               workqueue.RateLimitingInterface
	recorder                record.EventRecorder
}

// NewController returns a new VPN mesh controller
func NewController(
	kubeclientset kubernetes.Interface,
	edgenetclientset clientset.Interface,
	nodecontributionInformer informers.NodeContributionInformer,
	linkname string,
) *Controller {
	utilruntime.Must(edgenetscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:           kubeclientset,
		edgenetclientset:        edgenetclientset,
		nodecontributionsLister: nodecontributionInformer.Lister(),
		nodecontributionsSynced: nodecontributionInformer.Informer().HasSynced,
		linkname:                linkname,
		workqueue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VPNMesh"),
		recorder:                recorder,
	}

	klog.V(4).Info("Setting up event handlers")
	nodecontributionInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: controller.enqueueNodeContribution,
			UpdateFunc: func(old, new interface{}) {
				controller.enqueueNodeContribution(new)
			},
			// TODO: Handle deletion (remove peer)
		},
	)

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.V(4).Info("Starting VPN mesh controller")

	klog.V(4).Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.nodecontributionsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.V(4).Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.V(4).Info("Started workers")
	<-stopCh
	klog.V(4).Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.V(4).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the VPNMesh resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	nc, err := c.nodecontributionsLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("nodecontribution '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	if nc.Spec.VPN == nil {
		klog.V(4).Infof("No VPN configuration specified for nodecontribution object %s", nc.Name)
		return nil
	}

	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("error while creating WG client: %s", err.Error())
	}

	publicKey, err := wgtypes.ParseKey(nc.Spec.VPN.PublicKey)
	if err != nil {
		return fmt.Errorf("error while parsing WG public key: %s", err.Error())
	}

	allowedIPs := make([]net.IPNet, 0)
	for _, s := range nc.Spec.VPN.Addresses {
		ip := net.ParseIP(s)
		var mask net.IPMask
		if ip.To4() != nil {
			mask = net.CIDRMask(32, 32)
		} else {
			mask = net.CIDRMask(128, 128)
		}
		allowedIPs = append(allowedIPs, net.IPNet{IP: ip, Mask: mask})
	}

	peerConfig := wgtypes.PeerConfig{
		AllowedIPs:        allowedIPs,
		PublicKey:         publicKey,
		Remove:            false,
		ReplaceAllowedIPs: true,
		UpdateOnly:        false,
	}

	deviceConfig := wgtypes.Config{
		Peers:        []wgtypes.PeerConfig{peerConfig},
		ReplacePeers: false,
	}

	err = client.ConfigureDevice(c.linkname, deviceConfig)
	if err != nil {
		return fmt.Errorf("error while configure WG device %s: %s", c.linkname, err.Error())
	}

	c.recorder.Event(nc, corev1.EventTypeNormal, PeerSynced, MessagePeerSynced)
	return nil
}

// enqueueNodeContribution takes a NodeContribution resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than VPNMesh.
func (c *Controller) enqueueNodeContribution(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}
