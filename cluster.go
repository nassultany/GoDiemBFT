package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/kevinburke/nacl/sign"
)

type JoinClusterMessage struct {
	Ip        string `json:"ip"`
	GrpcPort  int    `json:"grpcPort"`
	NodePort  int    `json:"nodePort"`
	PublicKey []byte `json:"publicKey"`
}

type SetupMessage struct {
	ValidatorId string            `json:"validatorId"`
	PublicKeys  map[string][]byte `json:"publicKeys"`
	Validators  map[string]string `json:"validators"`
}

type NodeInfo struct {
	ip          string
	grpcPort    int
	nodePort    int
	publicKey   []byte
	validatorId string
}

func createCluster(lis net.Listener, publicKey sign.PublicKey, grpcPort, f int) (string, map[string]string, map[string][]byte) {
	var Nodes []NodeInfo
	id := 1
	publicKeys := make(map[string][]byte)
	validators := make(map[string]string)
	validator_id := "Validator_" + strconv.Itoa(id)
	publicKeys[validator_id] = publicKey
	myIps, _ := net.InterfaceAddrs()
	ip := strings.Split(myIps[0].String(), "/")[0]
	validators[validator_id] = fmt.Sprintf("%s:%d", ip, grpcPort)
	id++
	for {
		connIn, err := lis.Accept()
		if err != nil {
			fmt.Println(err)
		} else {
			var receivedMessage JoinClusterMessage
			json.NewDecoder(connIn).Decode(&receivedMessage)
			fmt.Printf("Received joinCluster request: %+v\n", receivedMessage)

			validatorId := "Validator_" + strconv.Itoa(id)
			id++
			Nodes = append(Nodes, NodeInfo{
				ip:          receivedMessage.Ip,
				grpcPort:    receivedMessage.GrpcPort,
				nodePort:    receivedMessage.NodePort,
				publicKey:   receivedMessage.PublicKey,
				validatorId: validatorId,
			})
			publicKeys[validatorId] = receivedMessage.PublicKey
			validators[validatorId] = fmt.Sprintf("%s:%d", receivedMessage.Ip, receivedMessage.GrpcPort)
			if len(validators) == 3*f+1 {
				fmt.Println("Got enough nodes, setting up cluster...")
				break
			}
		}
	}
	for _, Node := range Nodes {
		connSetup, err := net.Dial("tcp", fmt.Sprintf("%s:%d", Node.ip, Node.nodePort))
		if err != nil {
			log.Fatalln(err)
		}
		json.NewEncoder(connSetup).Encode(&SetupMessage{
			ValidatorId: Node.validatorId,
			PublicKeys:  publicKeys,
			Validators:  validators,
		})
	}
	fmt.Printf("My setup: ValidatorId: %s, Validators: %+v, PublicKeys: %+v\n", validator_id, validators, publicKeys)
	return validator_id, validators, publicKeys
}

func joinCluster(clusterAddr string, lis net.Listener, publicKey sign.PublicKey, nodePort, grpcPort int) (string, map[string]string, map[string][]byte) {
	connCluster, err := net.Dial("tcp", clusterAddr)
	if err != nil {
		log.Fatalln(err)
	}
	myIps, _ := net.InterfaceAddrs()
	ip := strings.Split(myIps[0].String(), "/")[0]
	json.NewEncoder(connCluster).Encode(&JoinClusterMessage{Ip: ip, GrpcPort: grpcPort, NodePort: nodePort, PublicKey: publicKey})
	fmt.Println("Waiting for setup message...")
	var setupMsg SetupMessage
	connIn, err := lis.Accept()
	if err != nil {
		log.Fatalln(err)
	}
	json.NewDecoder(connIn).Decode(&setupMsg)
	fmt.Printf("My setup: %+v\n", setupMsg)
	return setupMsg.ValidatorId, setupMsg.Validators, setupMsg.PublicKeys
}
