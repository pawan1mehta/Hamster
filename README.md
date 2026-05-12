# 🐹 Hamster
Hamster is a distributed job and task scheduler that manages the execution of workloads(raw binaries, scripts and container images) across a cluster of worker nodes.

# Design

As an initial design, we will start with the following approach. 
Based on further analysis, the design may evolve over time.

- Input: HAMSTER will support two workload formats — container images 
and binaries (scripts/executables).

- Control Plane: HAMSTER will have a single control plane. Initially it 
will be built as one service. As requirements become clearer, we will 
analyze and decouple it into multiple services where needed.

- Worker Nodes: There will be multiple worker nodes in the cluster. 
Each worker node is responsible for executing the assigned workloads.
