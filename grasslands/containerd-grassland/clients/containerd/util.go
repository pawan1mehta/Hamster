package containerd

import (
	ctd "github.com/containerd/containerd"
	"github.com/containerd/errdefs"
)

func isNotFound(err error) bool {
	return err != nil && errdefs.IsNotFound(err)
}

func isAlreadyExists(err error) bool {
	return err != nil && errdefs.IsAlreadyExists(err)
}

func toLocalState(state ctd.Status) ContainerState {
	switch state.Status {
	case ctd.Running:
		return RUNNING
	case ctd.Stopped:
		return EXITED
	case ctd.Created, ctd.Paused, ctd.Pausing:
		return CREATED
	default:
		return UNKNOW
	}
}
