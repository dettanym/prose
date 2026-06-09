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

1. Commit and push changes to the main branch of this repo.
2. Flux will watch for changes to the main branch and automatically deploy them
   [every half an hour](./evaluation/kubernetes/flux/config/cluster.yaml#L8).
3. To deploy them immediately, run `task cluster:reconcile`, which is defined
   [here](./.taskfiles/cluster/tasks.yml#L19)
4. TODO: Describe using helm / flux suspend to debug.
