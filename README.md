# argocd-agent

`argocd-agent` implements a hub/spoke architecture with a central control plane for the popular GitOps tool
[Argo CD](https://github.com/argoproj/argo-cd). It allows to scale out Argo CD in many-cluster scenarios by moving compute intensive parts of Argo CD (application controller, repository server) to the managed clusters, while keeping the control and observe components (API and UI) in a central location.

Some might refer to this architecture as the "pull model" with a "single pane of glass".

## Architecture

`argocd-agent` consists of two basic components, which resemble a client/server model:

* The *control plane*, which also hosts the Argo CD API server and some other requirements
* One or more *agents*

The *control plane* represents a central location that implements central management and observability, e.g. the Argo CD API and UI components. However, no reconciliation of Applications happens on the control plane.

An *agent* is deployed to each managed cluster. These clusters, however, are not connected from the control plane, like they would have been in the classical Argo CD multi-cluster setup. Instead, a subset of Argo CD (the application-controller, the applicationset-controller and the repo-server) is deployed to those servers as well. Depending on its configuration, the role of the agent is to either:

* Submit status information from the Applications on the managed cluster back to the control plane,
* Receive updates to Application configuration from the control plane
* A combination of above tasks

In all cases, it's the agent that initiates the connection to the control plane.

## Design principles

The following paragraphs describe the design principles upon which `argocd-agent` is built. 

### A permanent network connection is neither expected nor required

It is understood that managed clusters can be everywhere: In your black-fibre connected data centres, across different cloud providers, in a car 

### Managed clusters are and will stay autonomous

The agent does not interfere with or augment the reconciliation process and it isn't required for the core functionality of Argo CD. 
### The initiating component is always the agent, not the control plane

Connections are established in one direction only: from the agent to the control plane.


## Status and current limitations

**Important notice:** `argocd-agent` is in its early stages and still under very active development. Until the first stable version (i.e. v1.0) is reached, users must expect breaking changes between minor releases (e.g. between v0.2.0 to v0.3.0).

You can check the
[roadmap](ROADMAP.md)
for things that we plan to work in the future.

As of now, the following limitations apply to `argocd-agent`:

* Because `argocd-agent` makes extensive use of bidirectional streams, a HTTP/2 connection between the agents and the server is a hard requirement. None of the current RPC libaries (gRPC, connectrpc) support HTTP/1.x. If you have any forward or reverse proxies in between who do not support HTTP/2, many features of `argocd-agent` will not work.