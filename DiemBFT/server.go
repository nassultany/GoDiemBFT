package DiemBFT

import (
	context "context"

	pb "github.com/nassultany/GoDiemBFT/protos"
)

type DiemRPCServer struct {
	pb.UnimplementedDiemRPCServer
	RpcCh chan<- interface{}
}

func (s *DiemRPCServer) ProposalMsg(ctx context.Context, P *pb.ProposalMessage) (*pb.ProposalResponse, error) {
	s.RpcCh <- ProposalMessageToStruct(P)
	return &pb.ProposalResponse{}, nil
}

func (s *DiemRPCServer) TimeoutMsg(ctx context.Context, M *pb.TimeoutMessage) (*pb.TimeoutResponse, error) {
	s.RpcCh <- TimeoutMessageToStruct(M)
	return &pb.TimeoutResponse{}, nil
}

func (s *DiemRPCServer) VoteMsg(ctx context.Context, M *pb.VoteMessage) (*pb.VoteResponse, error) {
	s.RpcCh <- VoteMessageToStruct(M)
	return &pb.VoteResponse{}, nil
}

// func main() {
// 	lis, err := net.Listen("tcp", ":8080")
// 	if err != nil {
// 		log.Fatalf("Error binding to port 8080: %v\n", err)
// 	}

// 	channel := make(chan interface{})
// 	val := Validator{rpcCh: channel}

// 	grpcServer := grpc.NewServer()
// 	pb.RegisterDiemRPCServer(grpcServer, &diemRPCServer{rpcCh: channel})

// 	go val.run()

// 	go func() {
// 		fmt.Println("Starting server on port 8080...")
// 		err = grpcServer.Serve(lis)
// 		if err != nil {
// 			log.Fatalln(err)
// 		}
// 	}()

// 	// we can make a client right here.
// 	conn, err := grpc.Dial(":8080", grpc.WithInsecure())
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	client := pb.NewDiemRPCClient(conn)

// 	for {
// 		var payload string
// 		fmt.Printf("Enter payload: ")
// 		in := bufio.NewReader(os.Stdin)
// 		payload, err := in.ReadString('\n')
// 		payload = payload[:len(payload)-1]
// 		if payload == "exit" {
// 			break
// 		}
// 		_, err = client.ProposalMsg(context.Background(), &pb.ProposalMessage{Block: &pb.Block{Id: "hello", Payload: payload}})
// 		if err != nil {
// 			log.Println(err)
// 		}

// 	}
// }
