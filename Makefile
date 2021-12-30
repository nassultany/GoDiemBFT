all: proto
	go build diem.go cluster.go

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative protos/rpc.proto

clean:
	rm protos/*.go
	rm diem