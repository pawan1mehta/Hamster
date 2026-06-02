As an initial design, we will go with the following approach. Based on further analysis, we may revise the design.

Input: We will support two workload formats as input — container image and binary.

HAMSTER will have one control plane consisting of one or more services. Based on requirements and analysis, we will decide how many services are needed and decouple them accordingly.

There will be multiple worker nodes. Each worker node will be responsible for executing the assigned workload.

# HAMSTER Control Plane Design:

The user provides a container image. On receiving it, the HAMSTER control plane schedules the container on an available worker node based on the user's requirements.

The user can also provide a raw binary of their program. On receiving it, the HAMSTER control plane schedules the binary on an available worker node.

`Note: As initial support, we will only support Go, Java, and Python binaries for raw binary execution.`

Hamster (Workload Unit): A Hamster is the unit of workload in the HAMSTER system. Every container image or raw binary submitted to the HAMSTER cluster is referred to as a Hamster.

Initially, there will be three core services in the control plane:

### hm-server
Receives incoming requests and orchestrates the process by routing it to the appropriate service.

### hm-scheduler
Responsible for scheduling workloads onto available worker nodes.

### hm-manager
Responsible for managing the core lifecycle and state of workloads.

# Initial Architecture
![Architecture](https://raw.githubusercontent.com/pawan1mehta/Hamster/main/Doc/Architecture/initial.drawio.svg.svg)
