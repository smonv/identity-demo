package main

import (
	"log"
	"net"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/ory-am/hydra/client"
	"github.com/ory-am/hydra/sdk"
	pb "github.com/tthanh/identity-demo/proto"
)

const (
	port = ":50051"
)

var (
	hydra *sdk.Client
)

type server struct{}

func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	id := "id"

	newClient := &client.Client{
		ID:           id,
		Name:         req.Username,
		Secret:       req.Password,
		RedirectURIs: []string{"uri"},
	}

	var err = hydra.Client.CreateClient(newClient)
	if err != nil {
		return nil, err
	}

	res := &pb.RegisterResponse{
		Id:       id,
		Username: req.Username,
	}

	return res, nil
}

func main() {
	var err error
	hydra, err = sdk.Connect(
		sdk.ClientID("tthanh"),
		sdk.ClientSecret("secret"),
		sdk.ClusterURL("https://localhost:4444"),
		sdk.SkipTLSVerify(),
	)
	if err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen on: %v", err)
	}

	s := grpc.NewServer()

	pb.RegisterIdentityServer(s, &server{})

	s.Serve(lis)

}
