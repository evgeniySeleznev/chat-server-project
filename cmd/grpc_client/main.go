package main

import (
	"context"
	"log"
	"time"

	"github.com/fatih/color"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	desc "github.com/evgeniySeleznev/chat-server-project/pkg/chat_v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	address = "localhost:50051"
	chatID  = 42
)

func main() {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Fatalf("failed to close connection: %v", err)
		}
	}()

	c := desc.NewChatV1Client(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.Create(ctx, &desc.CreateRequest{Usernames: []string{"Alan", "Browdi"}})
	if err != nil {
		log.Fatalf("failed to get note by id: %v", err)
	}

	log.Printf(color.RedString("Chat create info:\n"), color.GreenString("%v", r.GetId()))

	_, err = c.SendMessage(ctx, &desc.SendMessageRequest{From: "Twitter", Text: "No idea", Timestamp: timestamppb.New(time.Now())})
	if err != nil {
		log.Fatalf("failed to send message: %v", err)
	}

	log.Printf(color.RedString("Send message info:\n"), color.GreenString("%v", "Twitter"))

	_, err = c.Delete(ctx, &desc.DeleteRequest{Id: chatID})
	if err != nil {
		log.Fatalf("failed to delete chat by id: %v", err)
	}

	log.Printf(color.RedString("Chat delete info:\n"), color.GreenString("%v", chatID))
}
