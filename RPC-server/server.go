package main

import (
	"bufio"
	context "context"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/nassultany/GoDiemBFT/protos"
	"google.golang.org/grpc"
)

type diemRPCServer struct {
	pb.UnimplementedDiemRPCServer
}

func (s *diemRPCServer) ProposalMsg(ctx context.Context, block *pb.Block) (*pb.ProposalResponse, error) {
	fmt.Println("Received Proposal Message: ", block)
	return &pb.ProposalResponse{}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error binding to port 8080: %v\n", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDiemRPCServer(grpcServer, &diemRPCServer{})

	go func() {
		fmt.Println("Starting server on port 8080...")
		err = grpcServer.Serve(lis)
		if err != nil {
			log.Fatalln(err)
		}
	}()

	// we can make a client right here.
	conn, err := grpc.Dial(":8080", grpc.WithInsecure())
	if err != nil {
		log.Println(err)
	}
	client := pb.NewDiemRPCClient(conn)

	for {
		var payload string
		fmt.Printf("Enter payload: ")
		in := bufio.NewReader(os.Stdin)
		payload, err := in.ReadString('\n')
		payload = payload[:len(payload)-1]
		if payload == "exit" {
			break
		}
		_, err = client.ProposalMsg(context.Background(), &pb.Block{Id: 2424, Round: 1, Payload: payload})
		if err != nil {
			log.Println(err)
		}

	}
}
