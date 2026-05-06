# app-discovery-operator
OpenShift Operator for automatic detection and labeling of Red Hat application pods.

## Description

This is currently in a Proof of Concept state. The operator identifies and labels JBoss EAP pods from multiple deployment methods:

- **Direct deployments**: Pods using Red Hat EAP images (e.g., `registry.redhat.io/jboss-eap-7/...`)
- **S2I builds**: Source-to-Image built applications with EAP base images
- **EAP Operator-managed pods**: Pods deployed via the EAP Operator (identified by `app.kubernetes.io/managed-by: eap-operator` label)

The operator adds the following labels to identified pods:
- `rht.comp`: Red Hat component/product name ("EAP")
- `rht.pod_image`: The pod's container image name
- `rht.pod_image_ver`: Version extracted from the pod's container image tag
- `rht.comp_discovered`: Unix timestamp of when the pod was first discovered

The product detection map in `identifier.go` can be expanded to include other Red Hat middleware products.

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### RBAC
Your operator will need to be run with the following permissions:

Get, List, Watch, Patch, Update on pods.

Get, List, Watch on images.

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
