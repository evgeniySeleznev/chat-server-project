package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/brianvoe/gofakeit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	desc "github.com/evgeniySeleznev/chat-server-project/pkg/chat_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

const grpcPort = 50051

type server struct {
	desc.UnimplementedChatV1Server
}

// Create ...
func (s *server) Create(ctx context.Context, req *desc.CreateRequest) (*desc.CreateResponse, error) {
	_ = ctx // <– подавляем линтер, без логических побочных эффектов
	log.Printf("Names: %v", req.GetUsernames())

	return &desc.CreateResponse{
		Id: gofakeit.Int64(),
	}, nil
}

// SendMessage ...
func (s *server) SendMessage(ctx context.Context, req *desc.SendMessageRequest) (*emptypb.Empty, error) {
	_ = ctx // <– подавляем линтер, без логических побочных эффектов
	log.Printf("Send message from: %s, text: %s, time: %v", req.GetFrom(), req.GetText(), req.GetTimestamp())
	return &emptypb.Empty{}, nil
}

// Delete ...
func (s *server) Delete(ctx context.Context, req *desc.DeleteRequest) (*emptypb.Empty, error) {
	_ = ctx // <– подавляем линтер, без логических побочных эффектов
	log.Printf("Delete request ID: %d", req.GetId())
	return &emptypb.Empty{}, nil
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	reflection.Register(s)
	desc.RegisterChatV1Server(s, &server{})

	log.Printf("server listening at %v", lis.Addr())

	if err = s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
