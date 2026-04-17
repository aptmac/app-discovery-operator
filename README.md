# app-discovery-operator
OpenShift Operator for automatic detection and labeling of Red Hat application pods.

## Description

This is currently in a Proof of Concept state. The only pods that will be identified and labelled are EAP at the moment, there's a map in `identifier.go` that can be expanded upon to include other images.

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### RBAC
Your operator will need to be run with the following permissions:

Get, List, Watch, Patch, Update on pods.

### Running on the cluster

You’ll need an OpenShift cluster to run against. You can use CRC to get a local cluster for testing, or run against a remote cluster. Note: Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster oc cluster-info shows).

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/app-discovery-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.


**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/app-discovery-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

### Undeploy controller

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```
