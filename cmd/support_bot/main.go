package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"tourism/internal/model"
	"tourism/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	if dbHost == "" {
		dbHost = "db"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	dsn := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + " password=" + dbPass + " dbname=" + dbName + " sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	supportToken := os.Getenv("SUPPORT_BOT_TOKEN")
	if supportToken == "" {
		log.Fatal("Не указан токен бота поддержки (SUPPORT_BOT_TOKEN)")
	}
	bot, err := tgbotapi.NewBotAPI(supportToken)
	if err != nil {
		log.Fatal("Ошибка инициализации support бота:", err)
	}
	log.Printf("Запущен бот поддержки %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		msg := update.Message
		chatID := msg.Chat.ID
		userTelegramID := msg.From.ID

		// Определяем пользователя и его роль
		user, err := userRepo.GetByTelegramID(userTelegramID)
		if err != nil {
			newUser := &model.User{
				TelegramID: userTelegramID,
				Username:   msg.From.UserName,
				FirstName:  msg.From.FirstName,
				LastName:   msg.From.LastName,
				Role:       "user",
			}
			id, _ := userRepo.Create(newUser)
			newUser.ID = id
			user = newUser
		}

		if msg.IsCommand() {
			switch msg.Command() {
			case "start":
				if user.Role == "support" {
					bot.Send(tgbotapi.NewMessage(chatID, "Оператор поддержки на связи. Ожидание обращений..."))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Здравствуйте! Опишите, пожалуйста, ваш вопрос, и оператор поддержки скоро ответит."))
				}
			case "answer":
				if user.Role != "support" {
					bot.Send(tgbotapi.NewMessage(chatID, "Команда недоступна."))
				} else {
					args := msg.CommandArguments()
					parts := strings.SplitN(args, " ", 2)
					if len(parts) < 2 {
						bot.Send(tgbotapi.NewMessage(chatID, "Использование: /answer <UserID> <текст ответа>"))
					} else {
						uid, err := strconv.Atoi(parts[0])
						if err != nil {
							bot.Send(tgbotapi.NewMessage(chatID, "Некорректный ID пользователя."))
						} else {
							replyText := parts[1]
							recipient, err := userRepo.GetByID(uid)
							if err != nil {
								bot.Send(tgbotapi.NewMessage(chatID, "Пользователь не найден."))
							} else {
								outMsg := tgbotapi.NewMessage(recipient.TelegramID, fmt.Sprintf("Ответ поддержки: %s", replyText))
								bot.Send(outMsg)
								messageRepo.Save(&model.Message{FromUserID: user.ID, ToUserID: recipient.ID, Content: replyText, IsSupport: true})
								bot.Send(tgbotapi.NewMessage(chatID, "Ответ отправлен пользователю."))
							}
						}
					}
				}
			}
			continue
		}

		// Обработка обычных сообщений
		if user.Role == "support" {
			bot.Send(tgbotapi.NewMessage(chatID, "Для ответа пользователю используйте команду /answer <ID> <сообщение>."))
		} else {
			messageRepo.Save(&model.Message{FromUserID: user.ID, Content: msg.Text, IsSupport: true})
			supportUsers := []model.User{}
			db.Select(&supportUsers, "SELECT * FROM users WHERE role='support'")
			if len(supportUsers) == 0 {
				bot.Send(tgbotapi.NewMessage(chatID, "Нет доступных операторов."))
			} else {
				for _, sup := range supportUsers {
					out := fmt.Sprintf("Запрос от пользователя %s (ID %d):\n%s", user.FirstName, user.ID, msg.Text)
					bot.Send(tgbotapi.NewMessage(sup.TelegramID, out))
				}
				bot.Send(tgbotapi.NewMessage(chatID, "Ваш запрос отправлен в службу поддержки. Ожидайте ответа."))
			}
		}
	}
}
