package server

import (
	"containerdgrassland/rpc"
	"context"
	"fmt"
	"net"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ContainerdGrasslandServer struct {
	logger *zap.Logger
	rpc.UnimplementedRuntimeServiceServer
}

type Config struct {
	Logger *zap.Logger

	Address string
}

func (server *ContainerdGrasslandServer) CreateContainer(context context.Context, request *rpc.CreateContainerRequest) (*rpc.CreateContainerResponse, error) {

	server.logger.Info("CreateContainer called!",
		zap.Any("request", request),
	)

	client, err := containerd.New(
		"/run/containerd/containerd.sock",
		containerd.WithDefaultNamespace("hamster"),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "connect containerd: %v", err)
	}
	defer client.Close()

	fmt.Println("connected to containerd successfully!")

	// Step 1: Pull
	image, err := client.Pull(context, request.GetConfig().GetImage().GetImage(), containerd.WithPullUnpack)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "pull image: %v", err)
	}

	name := request.GetConfig().GetMetadat().GetName()
	
	cmd := request.GetConfig().GetCommand()
	args := request.GetConfig().GetArgs()

	full := append([]string{}, cmd...)
	full = append(full, args...)

	// Step 2: Create container
	container, err := client.NewContainer(
		context,
		name,
		containerd.WithNewSnapshot(name+"-snap", image),
		containerd.WithNewSpec(
			oci.WithImageConfig(image),
			oci.WithProcessArgs(full...),
		),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "new container: %v", err)
	}

	// Step 3: Create task
	task, err := container.NewTask(context, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "new task: %v", err)
	}

	if err := task.Start(context); err != nil {
		task.Delete(context)
		return nil, status.Errorf(codes.Internal, "start task: %v", err)
	}

	//server.logger.Info("Container started successfully")
	//
	//// Step 5: Watch for exit
	//exitCh, _ := task.Wait(context)
	//_ = <-exitCh

	return nil, nil
}

func StartContainerdGrasslandServer(config *Config) {
	ContainerGrasslandAddress := config.Address

	list, err := net.Listen("tcp", ":"+ContainerGrasslandAddress)
	if err != nil {
		config.Logger.Error("Failed to listen",
			zap.Error(err),
		)
		return
	}

	server := grpc.NewServer()

	srv := &ContainerdGrasslandServer{
		logger: config.Logger,
	}

	rpc.RegisterRuntimeServiceServer(server, srv)

	config.Logger.Info("ContainerdGrasslandServer started!",
		zap.Any("address", list.Addr()),
	)

	if err := server.Serve(list); err != nil {
		config.Logger.Error("Failed to server",
			zap.Error(err),
		)
	}
}
