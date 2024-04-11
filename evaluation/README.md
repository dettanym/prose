# Evaluation with sample kubernetes cluster

## Install prerequisites

1. Install [Homebrew] in a MacOS or Linux machine:

   `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`

   (TODO: Convert this into a VM.)

2. Use Homebrew to install [Minikube](https://formulae.brew.sh/formula/minikube), [go-task](https://taskfile.dev/installation/#homebrew), [helm](https://helm.sh/docs/intro/install/#from-homebrew-macos) and [Flux](https://fluxcd.io/flux/installation/#install-the-flux-cli):

   `brew install minikube go-task helm fluxcd/tap/flux`

## Provision / Bootstrap the cluster

1. Run `minikube start` to create a K8s node.
   Pass `--subnet=''` flag to set a subnet for a cluster (>1 nodes).

2. Run `task cluster:verify` to check that all dependencies are setup. Use `brew update` to check new versions of any dependencies. Use `brew upgrade` to upgrade dependencies.

3. Run `task cluster:install` to bootstrap minikube with flux andload cluster config/info into bootstrapped minikube cluster.

4. You'd have your cluster behind your domain. In this repo, since we don't have our own public domain, we want our K8s DNS gateway to resolve requests for our domain (`my-example.com`) to the K8s DNS gateway's external IP address, which can be found using `kubectl get svc -n networking k8s-gateway` (under `EXTERNAL-IP`). Here's a link to the Arch wiki for [configuring DNSMASQD](https://wiki.archlinux.org/title/NetworkManager#Custom_dnsmasq_configuration). For example, my DNSMASQD configuration is as follows.

   ```
   # Forward queries for 'my-example.com' and all of its subdomains to
   # k8s-gateway running in minikube at 192.168.49.20
   server=/my-example.com/192.168.49.20
   ```

   Replace the IP address with the K8s gateway's external IP address found using the command above. _You can continue to browse the web as usual; this configuration simply ensures that my-example.com is resolved by your local DNS resolver._ This configuration file can be removed once you've turned minikube off.

5. Be happy! [Work on your cluster](../README.md#work-on-the-cluster).
