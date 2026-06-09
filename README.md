# prose

PRivacy ObServability and Enforcement Frameworks

## Evaluation with sample kubernetes cluster

See [evaluation/README.md](./evaluation/README.md) for details on running
evaluations.

## Contribution guide

Contribution guide can be found in [CONTRIBUTING.md](./CONTRIBUTING.md)

## Repository structure

```sh
📁 privacy-profile-composer # Main components of the Prose suite
📁 presidio                 # NLP service for analysing JSON payloads
📁 charts
└─📁 prose                  # Chart for installing Prose into a cluster
📁 evaluation
└─📁 kubernetes             # Kubernetes cluster defined as code
  ├─📁 bootstrap            # Flux installation
  ├─📁 flux                 # Main Flux configuration of repository
  └─📁 apps                 # Apps deployed into the cluster grouped by namespace
    ├─📁 default            # Default namespace
    └─📁 sockshop           # Sockshop app namespace
```
