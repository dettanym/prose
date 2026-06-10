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

## Setup cluster

Minikube cluster with the default configuration can be created using
`./evaluation/scripts/start-minikube-cluster.sh`. This script only needs to run
once, when creating cluster. If the cluster is destroyed, we can rerun this
script to get the cluster created again.

To work with minikube, we can use a small proxy script
`./evaluation/scripts/minikube.sh`. This script can be used to destroy the
cluster with `./evaluation/scripts/minikube.sh delete`.

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
