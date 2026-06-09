package containerd

import (
	"containerdgrassland/rpc"
	"context"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	"go.uber.org/zap"
)

type Client struct {
	containerdClient *containerd.Client

	mu *sync.Mutex

	tasks map[string]*taskInfo

	logger *zap.Logger
}

type Image struct {
	Image string
}

type Container struct {
	Name  string
	State ContainerState
	Pid   uint32
}

type ContainerState int32

const (
	CREATED ContainerState = 0
	RUNNING ContainerState = 1
	EXITED  ContainerState = 2
	UNKNOW  ContainerState = 3
)

func NewContainerdClient(logger *zap.Logger) (*Client, error) {
	logger.Info("Initializing the ContainerdClient!")

	client, err := containerd.New(
		"/run/containerd/containerd.sock",
		containerd.WithDefaultNamespace("hamster"),
	)
	if err != nil {
		logger.Error("couldn't create containerd client", zap.Error(err))
		return nil, err
	}
	fmt.Println("connected to containerd successfully!")

	return &Client{
		containerdClient: client,
		logger:           logger,
		mu:               &sync.Mutex{},
		tasks:            make(map[string]*taskInfo),
	}, nil
}

func (c *Client) PullImage(ctx context.Context, image *Image) (containerd.Image, error) {
	if image.Image == "" {
		return nil, fmt.Errorf("pull image: image reference is empty")
	}

	containerImage, err := c.containerdClient.Pull(ctx, image.Image, containerd.WithPullUnpack)
	if err != nil {
		return nil, fmt.Errorf("pull image %q: %w", image.Image, err)
	}

	return containerImage, nil
}

func (c *Client) CreateContainer(ctx context.Context, cfg *rpc.ContainerConfig) (*Container, error) {
	// Step 1: Pull
	imageRef := &Image{
		Image: cfg.GetImage().GetImage(),
	}

	containerImage, err := c.PullImage(ctx, imageRef)
	if err != nil {
		c.logger.Error("failed to pull image",
			zap.String("image", imageRef.Image),
			zap.Error(err))
		return nil, err
	}

	// Build process args
	var processArgs []string
	if len(cfg.GetCommand()) > 0 {
		processArgs = append(processArgs, cfg.GetCommand()...)
		processArgs = append(processArgs, cfg.GetArgs()...)
	}

	// Build env map
	var env []string
	for _, kv := range cfg.GetEnvs() {
		env = append(env, kv.GetKey()+"="+string(kv.GetValue()))
	}

	// Build OCI spec options
	specOpts := []oci.SpecOpts{
		oci.WithImageConfig(containerImage),
	}

	// Args
	if len(processArgs) > 0 {
		specOpts = append(specOpts, oci.WithProcessArgs(processArgs...))
	}

	if len(env) > 0 {
		specOpts = append(specOpts, oci.WithEnv(env))
	}

	// Working directory
	if wd := cfg.GetWorkingDir(); wd != "" {
		specOpts = append(specOpts, oci.WithProcessCwd(wd))
	}

	// Resources
	if res := cfg.GetResources(); res != nil {
		if res.GetMemoryLimitBytes() > 0 {
			specOpts = append(specOpts, oci.WithMemoryLimit(uint64(res.GetMemoryLimitBytes())))
		}

		const defaultCPUPeriod = 100000
		if res.GetCpuLimit() > 0 {
			quota := int64(res.GetCpuLimit() * float64(defaultCPUPeriod))
			specOpts = append(specOpts, oci.WithCPUCFS(quota, defaultCPUPeriod))
		}
		if res.GetPidsLimit() > 0 {
			specOpts = append(specOpts, oci.WithPidsLimit(res.GetPidsLimit()))
		}
	}

	// Mounts
	for _, m := range cfg.GetMounts() {
		opts := []string{"rbind"}
		if m.GetReadonly() {
			opts = append(opts, "ro")
		}

		specOpts = append(specOpts, oci.WithMounts([]specs.Mount{
			{
				Type:        m.GetType(),
				Source:      m.GetSource(),
				Destination: m.GetDestination(),
				Options:     opts,
			},
		}))
	}

	// Network
	if net := cfg.GetNetwork(); net != nil {
		switch net.GetMode() {
		case rpc.NetworkMode_NETWORK_HOST:
			specOpts = append(specOpts, oci.WithHostNamespace(specs.NetworkNamespace))
		case rpc.NetworkMode_NETWORK_NONE:
			// containerd/runc creates empty netns(network namespace) by default
		}
	}

	// Security
	if sec := cfg.GetSecurity(); sec != nil {
		if sec.GetReadonlyRootfs() {
			specOpts = append(specOpts, oci.WithRootFSReadonly())
		}
		if sec.GetNoNewPrivileges() {
			specOpts = append(specOpts, oci.WithNoNewPrivileges)
		}
		if sec.GetRunAsUser() != 0 || sec.GetRunAsGroup() != 0 {
			specOpts = append(specOpts, oci.WithUIDGID(
				uint32(sec.GetRunAsUser()),
				uint32(sec.GetRunAsGroup()),
			))
		}
	}

	name := cfg.GetMetadata().GetName()
	snapshotName := name + "-snap"

	_, err = c.containerdClient.NewContainer(
		ctx,
		name,
		containerd.WithNewSnapshot(snapshotName, containerImage),
		containerd.WithNewSpec(specOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("create container error: %v", err)
	}

	return &Container{
		Name: name,
	}, nil
}

func (c *Client) StartContainer(ctx context.Context, id string) error {

	// find the container
	container, err := c.containerdClient.LoadContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("container not found")
	}

	// check if already running
	if _, err := container.Task(ctx, nil); err == nil {
		return fmt.Errorf("container %q already running", id)
	}

	ioCreator := cio.NewCreator(cio.WithStdio)

	task, err := container.NewTask(ctx, ioCreator)
	if err != nil {
		return fmt.Errorf("new task: %v", err)
	}

	if err := task.Start(ctx); err != nil {
		task.Delete(ctx)
		return fmt.Errorf("start task: %v", err)
	}

	exitCh, err := task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("task wait: %w", err)
	}

	go func() {
		exitStatus := <-exitCh
		code, _, _ := exitStatus.Result()

		c.mu.Lock()
		if info, ok := c.tasks[id]; ok {
			info.exitCode = int(code)
		}
		c.mu.Unlock()

		c.logger.Info("container exited",
			zap.String("id", id),
			zap.Int32("code", int32(code)),
		)
	}()

	c.mu.Lock()
	c.tasks[id] = &taskInfo{
		task:   task,
		exitCh: exitCh,
	}
	c.mu.Unlock()

	return nil
}

func (c *Client) StopContainer(ctx context.Context, id string, timeoutSec time.Duration) error {
	container, err := c.containerdClient.LoadContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("load container %q: %w", id, err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return nil
	}

	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return fmt.Errorf("sigterm: %w", err)
	}

	timeout := time.Duration(timeoutSec) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second // default grace period
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	exitCh, err := task.Wait(ctx)
	if err != nil {
		return err
	}

	select {
	case <-exitCh:
		// exited cleanly after SIGTERM
	case <-waitCtx.Done():
		_ = task.Kill(ctx, syscall.SIGKILL)
		<-exitCh
	}

	return nil
}

func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	container, err := c.containerdClient.LoadContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("container %q doesn't exit", id)
	}

	if _, err := container.Task(ctx, nil); err == nil {
		return fmt.Errorf("container %q still running; stop it first", id)
	}

	err = container.Delete(ctx, containerd.WithSnapshotCleanup)
	if err != nil {
		c.logger.Error("Couldn't delete the container", zap.String("id", id))
		return err
	}

	return nil
}

func (c *Client) ContainerStatus(ctx context.Context, id string) (*Container, error) {
	container, err := c.containerdClient.LoadContainer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("container %q doesn't exit", id)
	}

	task, err := container.Task(ctx, nil)
	if err == nil {
		return &Container{
			Name:  id,
			State: RUNNING,
		}, nil
	}

	pid := task.Pid()

	return &Container{
		Name:  id,
		State: CREATED,
		Pid:   pid,
	}, nil
}
