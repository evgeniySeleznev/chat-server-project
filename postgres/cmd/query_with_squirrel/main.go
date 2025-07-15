package main

import (
	"context"
	"log"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/brianvoe/gofakeit"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	dbDSN = "host=localhost port=54322 dbname=chat-server-pg user=chat-server-user password=chat-server-password sslmode=disable"
)

func main() {
	ctx := context.Background()

	// Подключение к пулу
	pool, err := pgxpool.Connect(ctx, dbDSN)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	// 1. Создание пользователя
	username := gofakeit.Username()
	query, args, err := psql.
		Insert("users").
		Columns("username").
		Values(username).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		log.Fatalf("build user insert: %v", err)
	}

	var userID int
	err = pool.QueryRow(ctx, query, args...).Scan(&userID)
	if err != nil {
		log.Fatalf("insert user: %v", err)
	}
	log.Printf("Created user '%s' with id %d", username, userID)

	// 2. Создание чата
	query = "INSERT INTO chats DEFAULT VALUES RETURNING id"
	args = []interface{}{}
	if err != nil {
		log.Fatalf("build chat insert: %v", err)
	}

	var chatID int
	err = pool.QueryRow(ctx, query, args...).Scan(&chatID)
	if err != nil {
		log.Fatalf("insert chat: %v", err)
	}
	log.Printf("Created chat with id %d", chatID)

	// 3. Привязка пользователя к чату
	query, args, err = psql.
		Insert("chat_users").
		Columns("chat_id", "user_id").
		Values(chatID, userID).
		ToSql()
	if err != nil {
		log.Fatalf("build chat_users insert: %v", err)
	}

	_, err = pool.Exec(ctx, query, args...)
	if err != nil {
		log.Fatalf("insert chat_users: %v", err)
	}
	log.Printf("Linked user %d to chat %d", userID, chatID)

	// 4. Вставка сообщения
	text := gofakeit.HipsterSentence(6)
	query, args, err = psql.
		Insert("messages").
		Columns("chat_id", "sender_id", "text").
		Values(chatID, userID, text).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		log.Fatalf("build message insert: %v", err)
	}

	var messageID int
	err = pool.QueryRow(ctx, query, args...).Scan(&messageID)
	if err != nil {
		log.Fatalf("insert message: %v", err)
	}
	log.Printf("Inserted message with id %d", messageID)

	// 5. Выборка сообщений
	query, args, err = psql.
		Select("m.id", "m.text", "m.timestamp", "u.username", "m.chat_id").
		From("messages m").
		Join("users u ON u.id = m.sender_id").
		OrderBy("m.timestamp DESC").
		Limit(10).
		ToSql()
	if err != nil {
		log.Fatalf("build select: %v", err)
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		log.Fatalf("query messages: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var msgText string
		var ts time.Time
		var sender string
		var cID int

		if err := rows.Scan(&id, &msgText, &ts, &sender, &cID); err != nil {
			log.Fatalf("scan message: %v", err)
		}

		log.Printf("Message #%d | chat: %d | from: %s | at: %v | text: %s", id, cID, sender, ts.Format(time.RFC3339), msgText)
	}
}
