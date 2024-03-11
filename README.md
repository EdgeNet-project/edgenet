<p align="center">
    <img src="assets/logos/edgenet_logos_2020_05_03/edgenet_logo_2020_05_03_w_text_300dpi_10pct.png" alt="Welcome to EdgeNet" width="400">
</p>

<!-- [![Go Report Card](https://goreportcard.com/badge/github.com/EdgeNet-project/edgenet)](https://goreportcard.com/report/github.com/EdgeNet-project/edgenet)
[![Build Status](https://github.com/EdgeNet-project/edgenet/actions/workflows/test_and_publish.yaml/badge.svg?branch=main)](https://github.com/EdgeNet-project/edgenet/actions/workflows/test_and_publish.yaml)
[![Coverage Status](https://coveralls.io/repos/github/EdgeNet-project/edgenet/badge.svg?branch=main)](https://coveralls.io/github/EdgeNet-project/edgenet?branch=main)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/EdgeNet-project/edgenet)](https://github.com/EdgeNet-project/edgenet/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/EdgeNet-project/edgenet.svg)](https://pkg.go.dev/github.com/EdgeNet-project/edgenet) -->

# What is EdgeNet?
EdgeNet is a free and open-source cloud orchestration software extension that brings industry-standard cloud software [Kubernetes](https://kubernetes.io/) to the network edge. It allows you to deploy applications to hundreds of nodes that are scattered across the internet, rather than just one or a small number of centralized data centers.

EdgeNet software is different from [EdgeNet-Testbed](https://www.edge-net.org) which is a globally distributed edge cloud running EdgeNet software. If you are interested in joining the testbed and helping non-profit scientific research we encourage you to [contribute a node](https://www.edge-net.org/pages/node-contribution.html).

This repository contains the source code and documentation for EdgeNet software.

# Features
To extend, and adapt Kubernetes into edge computing, EdgeNet employs various features. 

<!-- You can click on them to go through [technical documentation](/docs/custom_resources.md) or see the [detailed concepts](/docs/README.md#concepts). -->
    
* [Multitenancy](/docs/README.md#multitenancy): EdgeNet enables the utilization of a shared cluster by multiple tenants who lack trust in each other. Tenants can allocate resource quotas or slices, and they also have the ability to offer their resources to other tenants. This functionality empowers tenants to function both as providers and consumers, operating in both vendor and consumer modes.

* [Multiprovider](/docs/README.md#multiprovider): By accommodating the collaboration of diverse providers, EdgeNet encourages numerous entities to contribute to nodes, thus fostering a rich and expansive ecosystem that thrives on heterogeneity. With the power of multitenancy, contributors with different hardware can easily lend their hardware.

* [Location-Based Node Selection](/docs/README.md#location-based-node-selection): While the involvement of multiple providers in EdgeNet extends beyond hardware vending, the possibilities encompass a broader spectrum. Node contributions can originate from individuals across the globe, and by leveraging a selective deployment mechanism, EdgeNet empowers the targeted deployment of resources to specific geographical regions, thereby augmenting localization capabilities and enabling efficient utilization of computing power where it is most needed.

* [Federation](/docs/README.md#federation-of-multiple-edgenet-clusters): EdgeNet envisions the federation of Kubernetes clusters worldwide, starting from the edge. By granting clusters the ability to assume workload or federation roles, EdgeNet enables the outsourcing of workloads to these clusters, fostering a seamless and globally interconnected network of distributed computing resources.

# Create an EdgeNet Cluster
To create an EdgeNet cluster you need to have admin access to a Kubernetes cluster. If you want to create one, you can see the Kubernetes cluster created with [minikube](https://kubernetes.io/docs/tutorials/kubernetes-basics/create-cluster/) or [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/) from the official documentation.

EdgeNet uses [cert-manager](https://cert-manager.io/docs/installation/) to manage certificates. If you don't have `cert-manager` in your Kubernetes cluster please install it first. 

To deploy EdgeNet features to your cluster refer to the [``advanced installation tutorial``](/docs/tutorials/deploy_edgenet_to_kube.md).

# Tutorials and Documentation
If you are planning to use EdgeNet software in your Kubernetes cluster, we highly encourage you to check out the [EdgeNet's documentation](/docs/README.md).

You can access all of [EdgeNet's tutorials](./docs/README.md#tutorials) with the main documentation or by navigating to the `doc` folder in the main repository.

# Support

To chat with a member of the EdgeNet team live, please [open our tawk.to window](https://tawk.to/edgenet).

# Contributing

The EdgeNet software is free and open source, licensed under the [Apache 2.0 license](https://www.apache.org/licenses/LICENSE-2.0); we invite you to contribute. You can access [contribution guide](/docs/guides/contribution_guides.md) for more information on how to contribute.


<!-- # edgenet
// TODO(user): Add simple overview of use/purpose

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/edgenet:tag
```

**NOTE:** This image ought to be published in the personal registry you specified. 
And it is required to have access to pull the image from the working environment. 
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/edgenet:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/edgenet:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/edgenet/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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
 -->
