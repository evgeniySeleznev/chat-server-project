package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/evgeniySeleznev/chat-server-project/internal/config"
	"github.com/evgeniySeleznev/chat-server-project/internal/config/env"
	"github.com/evgeniySeleznev/chat-server-project/internal/configenv"
	"log"
	"net"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	desc "github.com/evgeniySeleznev/chat-server-project/pkg/chat_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

type server struct {
	desc.UnimplementedChatV1Server
	pool *pgxpool.Pool
}

// Create ...
func (s *server) Create(ctx context.Context, req *desc.CreateRequest) (*desc.CreateResponse, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	userIDs := make([]int64, 0, len(req.Usernames))

	// 1. Вставляем или получаем пользователей
	for _, username := range req.Usernames {
		// Пытаемся вставить пользователя
		query, args, err := psql.Insert("users").
			Columns("username").
			Values(username).
			Suffix("ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username RETURNING id").
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("failed to build insert user query: %w", err)
		}

		var userID int64
		err = tx.QueryRow(ctx, query, args...).Scan(&userID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert or get user: %w", err)
		}

		userIDs = append(userIDs, userID)
	}

	// 2. Вставляем чат
	query, args, err := psql.Insert("chats").
		Columns().
		Values().
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert chat query: %w", err)
	}

	var chatID int64
	err = tx.QueryRow(ctx, query, args...).Scan(&chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert chat: %w", err)
	}

	// 3. Добавляем пользователей в чат
	for _, userID := range userIDs {
		query, args, err := psql.Insert("chat_users").
			Columns("chat_id", "user_id").
			Values(chatID, userID).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("failed to build insert chat_user query: %w", err)
		}

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to insert chat_user: %w", err)
		}
	}

	// 4. Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &desc.CreateResponse{
		Id: chatID,
	}, nil
}

// SendMessage ...
func (s *server) SendMessage(ctx context.Context, req *desc.SendMessageRequest) (*emptypb.Empty, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Найти ID пользователя по username
	query, args, err := psql.
		Select("id").
		From("users").
		Where(sq.Eq{"username": req.GetFrom()}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select user query: %w", err)
	}

	var userID int64
	err = tx.QueryRow(ctx, query, args...).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("sender not found: %w", err)
	}

	// 2. Найти ID чата, в котором состоит пользователь (например, последний созданный)
	query, args, err = psql.
		Select("chat_id").
		From("chat_users").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("chat_id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select chat query: %w", err)
	}

	var chatID int64
	err = tx.QueryRow(ctx, query, args...).Scan(&chatID)
	if err != nil {
		return nil, fmt.Errorf("chat for sender not found: %w", err)
	}

	// 3. Вставить сообщение
	query, args, err = psql.
		Insert("messages").
		Columns("chat_id", "sender_id", "text", "timestamp").
		Values(chatID, userID, req.GetText(), req.GetTimestamp().AsTime()).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert message query: %w", err)
	}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to insert message: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return &emptypb.Empty{}, nil
}

// Delete ...
func (s *server) Delete(ctx context.Context, req *desc.DeleteRequest) (*emptypb.Empty, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.
		Delete("chats").
		Where(sq.Eq{"id": req.GetId()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build delete query: %w", err)
	}

	res, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute delete query: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("chat with id %d not found", req.GetId())
	}

	return &emptypb.Empty{}, nil
}

func main() {
	flag.Parse()
	ctx := context.Background()

	// Считываем переменные окружения
	err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	grpcConfig, err := env.NewGRPCConfig()
	if err != nil {
		log.Fatalf("failed to get grpc config: %v", err)
	}

	pgConfig, err := env.NewPGConfig()
	if err != nil {
		log.Fatalf("failed to get pg config: %v", err)
	}

	lis, err := net.Listen("tcp", grpcConfig.Address())
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Создаем пул соединений с базой данных
	pool, err := pgxpool.Connect(ctx, pgConfig.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	s := grpc.NewServer()
	reflection.Register(s)
	desc.RegisterChatV1Server(s, &server{pool: pool})

	log.Printf("server listening at %v", lis.Addr())

	if err = s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
