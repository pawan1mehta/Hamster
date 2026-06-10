package server

import (
	"containerdgrassland/clients/containerd"
	"containerdgrassland/rpc"
)

func validateCreateContainerConfig(config *rpc.ContainerConfig) {

}

func validateStartContainerRequest(request *rpc.StartContainerRequest) {

}

func toRPCState(s containerd.ContainerState) rpc.ContainerState {
	switch s {
	case containerd.CREATED:
		return rpc.ContainerState_CONTAINER_CREATED
	case containerd.RUNNING:
		return rpc.ContainerState_CONTAINER_RUNNING
	case containerd.EXITED:
		return rpc.ContainerState_CONTAINER_EXITED
	case containerd.UNKNOW:
		return rpc.ContainerState_CONTAINER_UNKNOWN
	default:
		return rpc.ContainerState_CONTAINER_UNKNOWN
	}
}
