package main

import (
	"crypto/rand"
	"flag"
	"fmt"

	//"log"
	"net"
	"strconv"

	"github.com/nassultany/GoDiemBFT/DiemBFT"
	pb "github.com/nassultany/GoDiemBFT/protos"
	"google.golang.org/grpc"

	"github.com/kevinburke/nacl/sign"

	log "github.com/sirupsen/logrus"
)

var logLevels map[string]log.Level = map[string]log.Level{
	"trace": log.TraceLevel,
	"info":  log.InfoLevel,
	"debug": log.DebugLevel,
	"warn":  log.WarnLevel,
	"error": log.ErrorLevel,
	"fatal": log.FatalLevel,
	"panic": log.PanicLevel,
}

func main() {
	// Other nodes connect to the master and wait to receive a cluster setup message
	clusterMaster := flag.Bool("master", false, "start node as cluster master")
	f := flag.Int("f", 1, "number of faulty replicas tolerated")
	grpcPort := flag.Int("grpcPort", 8080, "port to listen for rpcs")
	nodePort := flag.Int("port", 8000, "port to listen for cluster and client configuration")
	clusterAddr := flag.String("clusterAddr", "127.0.0.1:8000", "ip address of cluster master node")
	logLevelString := flag.String("logLevel", "info", "log level: trace, debug, info, warning, error, fatal, panic")
	flag.Parse()

	var logLevel log.Level
	logLevel, ok := logLevels[*logLevelString]
	if !ok {
		log.Info("Invalid log level specified. Defaulting to level=info.")
		logLevel = log.InfoLevel
	}

	var validators map[string]string
	var publicKeys map[string][]byte
	var validatorId string

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(*nodePort))
	if err != nil {
		log.Fatalf("Error binding to port %d: %v\n", *grpcPort, err)
	}

	publicKey, privateKey, err := sign.Keypair(rand.Reader)
	if err != nil {
		log.Fatalln(err)
	}

	lisGrpc, err := net.Listen("tcp", ":"+strconv.Itoa(*grpcPort))
	if err != nil {
		log.Fatalf("Error binding to port %d: %v\n", *grpcPort, err)
	}
	rpcCh := make(chan interface{}, 1)

	// gRPC receive handlers will pass RPC calls to the validator through rpcCh
	grpcServer := grpc.NewServer()
	pb.RegisterDiemRPCServer(grpcServer, &DiemBFT.DiemRPCServer{RpcCh: rpcCh})

	go func() {
		fmt.Printf("Starting grpc server on port %d...\n", *grpcPort)
		err = grpcServer.Serve(lisGrpc)
		if err != nil {
			log.Fatalln(err)
		}
	}()

	if *clusterMaster {
		validatorId, validators, publicKeys = createCluster(lis, publicKey, *grpcPort, *f)
	} else {
		validatorId, validators, publicKeys = joinCluster(*clusterAddr, lis, publicKey, *nodePort, *grpcPort)
	}
	lis.Close()

	val := DiemBFT.NewValidator(validatorId, *f, validators, rpcCh, privateKey, publicKeys, logLevel)

	val.Run()
}
