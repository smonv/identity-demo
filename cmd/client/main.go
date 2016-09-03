package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"

	pb "github.com/tthanh/identity-demo/proto"
)

const (
	address = "localhost:50051"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	iClient := pb.NewIdentityClient(conn)

	args := os.Args[1:]

	if args[0] == "register" {
		username := args[1]
		password := args[2]

		req := &pb.RegisterRequest{
			Username: username,
			Password: password,
		}

		res, err := iClient.Register(context.Background(), req)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%v\n", res.Id)
		fmt.Printf("%v\n", res.Username)
	}
}
