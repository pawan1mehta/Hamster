package server

import (
	containerdclient "containerdgrassland/clients/containerd"
	"containerdgrassland/rpc"
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ContainerdGrasslandServer struct {
	logger *zap.Logger

	containerdClient *containerdclient.Client

	rpc.UnimplementedRuntimeServiceServer
}

type Config struct {
	Logger *zap.Logger

	Address string
}

func (server *ContainerdGrasslandServer) CreateContainer(context context.Context, request *rpc.CreateContainerRequest) (*rpc.CreateContainerResponse, error) {
	server.logger.Info("Creating a container...")

	cfg := request.GetConfig()

	// TODO: valid the create container config
	validateCreateContainerConfig(cfg)

	container, err := server.containerdClient.CreateContainer(context, cfg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create container: %v", err)
	}

	server.logger.Info("Container created successfully!")

	return &rpc.CreateContainerResponse{
		ContainerId: container.Name,
	}, nil
}

func (server *ContainerdGrasslandServer) StartContainer(context context.Context, request *rpc.StartContainerRequest) (*rpc.StartContainerResponse, error) {
	id := request.GetContainerId()

	server.logger.Info("Starting the container",
		zap.String("containerId", id),
	)

	// TODO: valid the start container request
	validateStartContainerRequest(request)

	if err := server.containerdClient.StartContainer(context, id); err != nil {
		return nil, status.Errorf(codes.Internal, "container start: %v", err)
	}

	server.logger.Info("Successfully started the container",
		zap.String("containerId", id),
	)

	return &rpc.StartContainerResponse{}, nil
}

func (server *ContainerdGrasslandServer) StopContainer(context context.Context, req *rpc.StopContainerRequest) (*rpc.StopContainerResponse, error) {
	id := req.GetContainerId()
	timeout := req.GetTimeoutSeconds()

	server.logger.Info("Stoping the container",
		zap.String("containerId", id),
	)

	err := server.containerdClient.StopContainer(context, id, time.Duration(timeout))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "couldn't sotp container: %v", err)
	}

	server.logger.Info("Successfully stopped the container",
		zap.String("containerId", id),
	)

	return &rpc.StopContainerResponse{}, nil
}

func (server *ContainerdGrasslandServer) RemoveContainer(context context.Context, req *rpc.RemoveContainerRequest) (*rpc.RemoveContainerResponse, error) {
	id := req.GetContainerId()

	server.logger.Info("Removing the container",
		zap.String("containerId", id),
	)

	err := server.containerdClient.RemoveContainer(context, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "couldn't remove container: %v", err)
	}

	server.logger.Info("Successfully removed the container",
		zap.String("containerId", id),
	)

	return &rpc.RemoveContainerResponse{}, nil
}

func (server *ContainerdGrasslandServer) ContainerStatus(context context.Context, req *rpc.ContainerStatusRequest) (*rpc.ContainerStatusResponse, error) {
	id := req.GetContainerId()

	server.logger.Info("Get container status",
		zap.String("containerId", id),
	)

	containerStatus, err := server.containerdClient.ContainerStatus(context, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "couldn't remove container: %v", err)
	}

	server.logger.Info("Successfully removed the container",
		zap.String("containerId", id),
	)

	return &rpc.ContainerStatusResponse{
		ContainerId: containerStatus.Name,
		State:       toRPCState(containerStatus.State),
		Pid:         containerStatus.Pid,
	}, nil
}

func StartContainerdGrasslandServer(config *Config) error {
	ContainerGrasslandAddress := config.Address

	list, err := net.Listen("tcp", ":"+ContainerGrasslandAddress)
	if err != nil {
		config.Logger.Error("Failed to listen",
			zap.Error(err),
		)
		return err
	}

	server := grpc.NewServer()

	containerdClient, err := containerdclient.NewContainerdClient(config.Logger)
	if err != nil {
		return fmt.Errorf("init containerd client: %w", err)
	}

	srv := &ContainerdGrasslandServer{
		logger:           config.Logger,
		containerdClient: containerdClient,
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

	return nil
}
