package main

import (
    "database/sql"
    "fmt"
    "log"

    _ "github.com/lib/pq"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const adminID int64 = // айди админа ПИСАТЬ

var db *sql.DB

func initDB() {
    var err error
    connStr := "user=ТВОЙ ПОЛЬЗОВАТЕЛЬ БАЗЫ ПОСТГРЕС password=ТВОЙ ПАРОЛЬ dbname=tg_bot_db sslmode=disable" // постгрес пользователь с паролем ПИСАТЬ
    db, err = sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    err = db.Ping()
    if err != nil {
        log.Fatal("Failed to ping database:", err)
    }
    fmt.Println("Database connected")

    // Создание таблицы, если она не существует
    createTableQuery := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        telegram_id BIGINT UNIQUE NOT NULL,
        username VARCHAR(255),
        first_name VARCHAR(255),
        last_name VARCHAR(255),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    `
    _, err = db.Exec(createTableQuery)
    if err != nil {
        log.Fatal("Failed to create table:", err)
    }
    fmt.Println("Table checked/created")
}

// Остальная часть кода остается без изменений

func registerUser(telegramID int64, username, firstName, lastName string) error {
    _, err := db.Exec(`INSERT INTO users (telegram_id, username, first_name, last_name)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (telegram_id) DO NOTHING`,
        telegramID, username, firstName, lastName)
    return err
}

func isUserRegistered(telegramID int64) (bool, error) {
    var exists bool
    err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id=$1)`, telegramID).Scan(&exists)
    return exists, err
}

func getAllUsers() ([]int64, error) {
    rows, err := db.Query(`SELECT telegram_id FROM users`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []int64
    for rows.Next() {
        var telegramID int64
        if err := rows.Scan(&telegramID); err != nil {
            return nil, err
        }
        users = append(users, telegramID)
    }
    return users, nil
}

func main() {
    initDB()
    defer db.Close()

    bot, err := tgbotapi.NewBotAPI("") //API TOKEN  ПИСАТЬ
    if err != nil {
        log.Panic(err)
    }

    bot.Debug = true
    log.Printf("Authorized on account %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			chatID := update.Message.Chat.ID
	
			// Проверка на ID администратора
			if chatID == adminID {
				users, err := getAllUsers()
				if err != nil {
					log.Printf("Failed to get users: %v", err)
					continue
				}
	
				for _, userID := range users {
					if userID != chatID { // Исключаем отправителя
						if update.Message.Text != "" {
							// Отправка текстового сообщения
							msg := tgbotapi.NewMessage(userID, update.Message.Text)
							bot.Send(msg)
						}
	
						if update.Message.Photo != nil {
							// Пересылка фото (берем самое большое изображение)
							photo := update.Message.Photo[len(update.Message.Photo)-1]
							msg := tgbotapi.NewPhoto(userID, tgbotapi.FileID(photo.FileID))
							bot.Send(msg)
						}
	
						if update.Message.Document != nil {
							// Пересылка документа
							doc := update.Message.Document
							msg := tgbotapi.NewDocument(userID, tgbotapi.FileID(doc.FileID))
							bot.Send(msg)
						}
	
						// Добавьте другие типы сообщений по аналогии, если нужно
					}
				}
			} else {
				msg := tgbotapi.NewMessage(chatID, "У вас нет прав на отправку сообщений всем пользователям.")
				bot.Send(msg)
			}
		}
	}
}
