# prose

PRivacy ObServability and Enforcement Frameworks

## Evaluation with sample kubernetes cluster

See [evaluation/README.md](./evaluation/README.md) for details on running evaluations.

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

## Work on the cluster

1. Commit and push changes to the main branch of this repo.
2. Flux will watch for changes to the main branch and automatically deploy them [every half an hour](./evaluation/kubernetes/flux/config/cluster.yaml#L8).
3. To deploy them immediately, run `task cluster:reconcile`, which is defined [here](./.taskfiles/cluster/tasks.yml#L19)
4. TODO: Describe using helm / flux suspend to debug.
