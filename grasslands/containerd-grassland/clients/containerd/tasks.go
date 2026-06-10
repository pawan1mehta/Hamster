package containerd

import (
	"github.com/containerd/containerd"
)

type taskInfo struct {
	task     containerd.Task
	exitCode int
	exitCh   <-chan containerd.ExitStatus
}
