package main

import (
	"context"
	"log"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/jackc/pgx/v4"
)

const (
	dbDSN = "host=localhost port=54322 dbname=chat-server-pg user=chat-server-user password=chat-server-password sslmode=disable"
)

func main() {
	ctx := context.Background()

	// Создаем соединение с базой данных
	conn, err := pgx.Connect(ctx, dbDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	//	// Делаем запрос на вставку записи в таблицу note
	//	res, err := con.Exec(ctx, "INSERT INTO note (title, body) VALUES ($1, $2)", gofakeit.City(), gofakeit.Address().Street)
	//	if err != nil {
	//		log.Fatalf("failed to insert note: %v", err)
	//	}
	//
	//	log.Printf("inserted %d rows", res.RowsAffected())
	//
	//	// Делаем запрос на выборку записей из таблицы note
	//	rows, err := con.Query(ctx, "SELECT id, title, body, created_at, updated_at FROM note")
	//	if err != nil {
	//		log.Fatalf("failed to select notes: %v", err)
	//	}
	//	defer rows.Close()
	//
	//	for rows.Next() {
	//		var id int
	//		var title, body string
	//		var createdAt time.Time
	//		var updatedAt sql.NullTime
	//
	//		err = rows.Scan(&id, &title, &body, &createdAt, &updatedAt)
	//		if err != nil {
	//			log.Fatalf("failed to scan note: %v", err)
	//		}
	//
	//		log.Printf("id: %d, title: %s, body: %s, created_at: %v, updated_at: %v\n", id, title, body, createdAt, updatedAt)
	//	}
	//}
	// Создание пользователя
	username := gofakeit.Username()
	var userID int
	err = conn.QueryRow(ctx, "INSERT INTO users (username) VALUES ($1) RETURNING id", username).Scan(&userID)
	if err != nil {
		log.Fatalf("failed to insert user: %v", err)
	}
	log.Printf("created user %s with id %d", username, userID)

	// Создание чата
	var chatID int
	err = conn.QueryRow(ctx, "INSERT INTO chats DEFAULT VALUES RETURNING id").Scan(&chatID)
	if err != nil {
		log.Fatalf("failed to insert chat: %v", err)
	}
	log.Printf("created chat with id %d", chatID)

	// Привязка пользователя к чату
	_, err = conn.Exec(ctx, "INSERT INTO chat_users (chat_id, user_id) VALUES ($1, $2)", chatID, userID)
	if err != nil {
		log.Fatalf("failed to insert into chat_users: %v", err)
	}
	log.Printf("linked user %d to chat %d", userID, chatID)

	// Вставка сообщения
	messageText := gofakeit.HipsterSentence(5)
	var messageID int
	err = conn.QueryRow(ctx,
		"INSERT INTO messages (chat_id, sender_id, text) VALUES ($1, $2, $3) RETURNING id",
		chatID, userID, messageText,
	).Scan(&messageID)
	if err != nil {
		log.Fatalf("failed to insert message: %v", err)
	}
	log.Printf("inserted message with id %d", messageID)

	// Выборка всех сообщений с привязкой к пользователям и чатам
	rows, err := conn.Query(ctx, `
		SELECT m.id, m.text, m.timestamp, u.username, m.chat_id
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		ORDER BY m.timestamp DESC
	`)
	if err != nil {
		log.Fatalf("failed to select messages: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var text, username string
		var chatID int
		var timestamp time.Time

		err := rows.Scan(&id, &text, &timestamp, &username, &chatID)
		if err != nil {
			log.Fatalf("failed to scan message: %v", err)
		}

		log.Printf("message_id: %d | user: %s | chat_id: %d | time: %v | text: %s",
			id, username, chatID, timestamp.Format(time.RFC3339), text)
	}
}
