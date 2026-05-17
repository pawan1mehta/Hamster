# Worker Node:

Since hamsters live in grasslands, each worker node will have one managing process called Grassland. Since there are two types of workloads — containers and raw binaries — each worker node will have the following Grassland runtimes: one for containers (powered by containerd) and one custom-made runtime environment for each supported language.

# Grasslands:

The Grassland process is responsible for managing the runtime lifecycle of a workload on its worker node.

- containerd-grassland: A runtime that manages the complete lifecycle of container workloads on the host system, powered by containerd.

- go-grassland: A runtime responsible for executing Go binary workloads.

- java-grassland: A runtime responsible for executing Java binary workloads.

- python-grassland: A runtime responsible for executing Python workloads.
