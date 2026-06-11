# Contributing to prose

## Development environment

Development environment is provisioned using nix. If you cannot install nix on
your machine natively, you can alternatively try to use
[nix-portable](github.com/DavHau/nix-portable).

Once nix is installed, scripts should pick up the nix binary and use development
tools from the current environment defined in this repository. If you need an
explicit access to development tools, then you can activate the development
environment with `nix develop` or `nix-portable nix develop`. This will make all
development tools available on the shell.

## Setup minikube

We use minikube with docker backend to run the cluster. If you are using
system-wide (root-based / rootful) docker, you can proceed to the next step. If
you are using rootless docker, make sure the configuration is complete. A good
article describing how to setup minikube with rootless docker can be found here:
https://wiki.archlinux.org/title/Minikube#Rootless_Docker.

## Setup the cluster

Minikube cluster with the default configuration can be created using
`./evaluation/scripts/start-minikube-cluster.sh`. This script only needs to run
once, when creating cluster. If the cluster is destroyed, we can rerun this
script to get the cluster created again.

To work with minikube, we can use a small proxy script
`./evaluation/scripts/minikube.sh`. This script can be used to destroy the
cluster with `./evaluation/scripts/minikube.sh delete`.

## Access the cluster

The cluster is running both workload containers for bookinfo sample application
and nginx ingress container to route all incoming requests to appropriate
workload containers. Depending on how we need the traffic to flow and the
requirements of deployed workload containers, we can access workload containers
directly or we can access them in a routed way through nginx ingress container.

To access workload containers directly, we need to have service kubernetes
resources with certain exposed ports existing in the cluster. To access through
nginx ingress, we need to direct traffic at nginx ingress service, but we also
need to supply enough information to nginx to be able to route that traffic. The
routing is specified by ingress kubernetes resources. There is a number of
ingress resources that exist for each bookinfo deployment.

### Performing queries through nginx ingress container

The easiest way to run queries against specified pages within the cluster is by
using curl with a specific `Host` header. This allows us to target a specific
ingress resource without also needing to trick our browser by substituting DNS
query results.

First, setup a proxy to the service using minikube functionality. We can see all
available services with target ports using `minikube service list`. Then the
proxy can be created with
`minikube service -n networking ingress-nginx-controller --url` command. It will
print number of URLs corresponding to the number of ports on the service. Find a
URL corresponding to `https/443` target port of nginx ingress service. In this
case, the returned value is `http://127.0.0.1:44513` and we will use this for
the rest of the example. The tunnel will be active while this command is running
and it can be interrupted if we hit `ctrl-C`.

If your cluster is running on your machine, you can curl this port directly. For
example,
`curl -k -H 'Host: jaeger-query.my-example.com' http://127.0.0.1:44513`.

If your cluster is running on the remote machine, you need to setup port
forwarding from your local machine to the remote machine. For example, this can
be achieved using ssh: `ssh -L 8443:127.0.0.1:44513 %remote-machine-host%`.
Substitute `%remote-machine-host%` with connection information to your remote
host. Again, the tunnel will be active while this ssh connection is active.
After this, from your local machine we can query through this forwarded port:
`curl -k -H 'Host: jaeger-query.my-example.com' http://127.0.0.1:8443`.

## Work on the cluster

The created cluster will contain a component called Flux. Flux will periodically
scan the remote github repository and reconcile the state of the local cluster
to match the resources defined on the remote repository. By default, only the
changes in the `main` branch are the ones that will be synchronized to your
cluster. It will deploy them
[every 30 minutes](./evaluation/kubernetes/flux/config/cluster.yaml#L8). In
order to trigger flux to immediately pull changes from the remote repository, we
can run `./evaluation/scripts/cluster-reconcile.sh` or
`nix run .#cluster:reconcile`. Flux can also be instructed to suspend certain
resource. This lets us change resources on the cluster and those changes will
not be reset by Flux next time it synchronizes with the remote repository. This
can be done by running `flux suspend kustomizaion <name>`. It is important to
remember to resume these resources after we are done with local changes. This
can be done using `flux resume kustomization <name>`.

The default configuration on the `main` branch will ensure that flux
synchronizes changes only from the `main` branch. In order for us to develop
using branches other the `main`, we can point flux to pull from an alternative
branch. The branch still needs to be present in the remote repository. Here are
the steps to achieve this:

1. Create a local branch from `main`:

   ```bash
   git switch -c feature/something-new main
   ```

2. Change the base branch name from `main` to `feature/something-new` in cluster
   configuration file
   [`./evaluation/kubernetes/flux/config/cluster.yaml`](./evaluation/kubernetes/flux/config/cluster.yaml).
   So it will become:

   ```yaml
   ref:
     branch: feature/something-new
   ```

3. Commit this change and push the branch to remote:

   ```bash
   git add ./evaluation/kubernetes/flux/config/cluster.yaml
   git commit
   git push --set-upstream origin feature/something-new
   ```

4. Change your local cluster to point to the new branch and reconcile your
   cluster:

   ```bash
   kubectl patch -n flux-system gitrepository prose-k8s-home-ops -p '{"spec":{"ref":{"branch":"feature/something-new"}}}'
   ./evaluation/scripts/cluster-reconcile.sh
   ```
