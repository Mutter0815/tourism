package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"tourism/internal/model"
	"tourism/internal/repository"
	"tourism/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// ReplyKeyboard для туристов
func userKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📍 Найти локации"),
			tgbotapi.NewKeyboardButton("🗺 Новый маршрут"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ Подписка на предложения"),
			tgbotapi.NewKeyboardButton("🛎 Поддержка"),
		),
	)
}

// ReplyKeyboard для провайдеров
func providerKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📍 Найти локации"),
			tgbotapi.NewKeyboardButton("📦 Мои бронирования"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🛎 Поддержка"),
		),
	)
}

// ReplyKeyboard для поддержки
func supportKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📤 Рассылка"),
			tgbotapi.NewKeyboardButton("📷 Добавить фото"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔍 Проверить локации"),
		),
	)
}

func main() {
	// Подключение к БД
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "db"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort,
		os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"),
	)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}

	// Репозитории и сервисы
	userRepo := repository.NewUserRepository(db)
	locationService := service.NewLocationService(repository.NewLocationRepository(db))
	tripService := service.NewTripService(repository.NewTripRepository(db), repository.NewLocationRepository(db))
	bookingService := service.NewBookingService(repository.NewBookingRepository(db))
	chatService := service.NewChatService(repository.NewBookingRepository(db), userRepo, repository.NewLocationRepository(db))
	offerService := service.NewOfferService(repository.NewSubscriptionRepository(db))
	authService := service.NewAuthService(userRepo)

	// Инициализация бота
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Не указан токен бота (BOT_TOKEN)")
	}
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Ошибка инициализации бота:", err)
	}
	log.Printf("Запущен бот %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Состояние
	activeTrip := make(map[int64]int)
	pendingBooking := make(map[int64]int)
	pendingAddPhoto := make(map[int64]int)

	for update := range updates {
		// Inline callback
		if cq := update.CallbackQuery; cq != nil {
			bot.Request(tgbotapi.NewCallback(cq.ID, ""))
			fromID := cq.From.ID
			data := cq.Data
			// обработка inline... (как было)
			continue
		}

		// Сообщения
		if update.Message == nil {
			continue
		}
		msg := update.Message
		chatID := msg.Chat.ID
		userID := msg.From.ID
		text := msg.Text

		// Команда /start с меню
		if msg.IsCommand() && msg.Command() == "start" {
			user, _ := authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
			var kb tgbotapi.ReplyKeyboardMarkup
			switch user.Role {
			case "provider":
				kb = providerKeyboard()
			case "support":
				kb = supportKeyboard()
			default:
				kb = userKeyboard()
			}
			resp := tgbotapi.NewMessage(chatID, fmt.Sprintf("Здравствуйте, %s! Выберите действие:", user.FirstName))
			resp.ReplyMarkup = kb
			bot.Send(resp)
			continue
		}

		// Menu buttons
		switch text {
		case "📍 Найти локации":
			bot.Send(tgbotapi.NewMessage(chatID, "Введите слово для поиска (или * для всех):"))
			continue

		case "🗺 Новый маршрут":
			user, _ := userRepo.GetByTelegramID(userID)
			if user == nil {
				user, _ = authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
			}
			tripID, err := tripService.CreateTrip(user.ID, "Мой маршрут")
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "Не удалось создать маршрут."))
			} else {
				activeTrip[userID] = tripID
				bot.Send(tgbotapi.NewMessage(chatID, "Маршрут создан. Добавляйте локации."))
			}
			continue

		case "✅ Подписка на предложения":
			user, _ := userRepo.GetByTelegramID(userID)
			if user == nil {
				user, _ = authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
			}
			offerService.Subscribe(user.ID)
			bot.Send(tgbotapi.NewMessage(chatID, "Вы подписаны на рассылку предложений."))
			continue

		case "🛎 Поддержка":
			bot.Send(tgbotapi.NewMessage(chatID, "Перейдите в @TouristSupportHelpBot для поддержки."))
			continue

		case "📦 Мои бронирования":
			bot.Send(tgbotapi.NewMessage(chatID, "Функция пока не реализована."))
			continue

		case "📤 Рассылка":
			user, _ := userRepo.GetByTelegramID(userID)
			if user.Role != "support" {
				bot.Send(tgbotapi.NewMessage(chatID, "Команда доступна только поддержке."))
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "Используйте /broadcast <текст> для рассылки."))
			}
			continue

		case "📷 Добавить фото":
			bot.Send(tgbotapi.NewMessage(chatID, "Используйте /addphoto <ID_локации> чтобы добавить фото."))
			continue

		case "🔍 Проверить локации":
			bot.Send(tgbotapi.NewMessage(chatID, "Функция пока не реализована."))
			continue
		}

		// Существующие команды
		if msg.IsCommand() {
			switch msg.Command() {
			case "locations":
				bot.Send(tgbotapi.NewMessage(chatID, "Введите слово для поиска (или * для всех):"))
			case "newtrip":
				user, _ := userRepo.GetByTelegramID(userID)
				if user == nil {
					user, _ = authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				tripID, err := tripService.CreateTrip(user.ID, "Мой маршрут")
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Не удалось создать маршрут."))
				} else {
					activeTrip[userID] = tripID
					bot.Send(tgbotapi.NewMessage(chatID, "Маршрут создан. Добавляйте локации."))
				}
			case "subscribe_offers":
				user, _ := userRepo.GetByTelegramID(userID)
				if user == nil {
					user, _ = authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				offerService.Subscribe(user.ID)
				bot.Send(tgbotapi.NewMessage(chatID, "Вы подписаны на рассылку предложений."))
			case "unsubscribe_offers":
				user, _ := userRepo.GetByTelegramID(userID)
				if user == nil {
					user, _ = authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				offerService.Unsubscribe(user.ID)
				bot.Send(tgbotapi.NewMessage(chatID, "Вы отписаны от рассылки."))
			case "addphoto":
				args := msg.CommandArguments()
				locID, err := strconv.Atoi(args)
				if args == "" || err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Используйте: /addphoto <ид_локации>"))
				} else {
					user, _ := userRepo.GetByTelegramID(userID)
					if user.Role != "support" {
						bot.Send(tgbotapi.NewMessage(chatID, "Нет прав для добавления фото."))
					} else {
						pendingAddPhoto[userID] = locID
						bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Отправьте фото для локации #%d", locID)))
					}
				}
			case "broadcast":
				user, _ := userRepo.GetByTelegramID(userID)
				if user.Role != "support" {
					bot.Send(tgbotapi.NewMessage(chatID, "Команда доступна только поддержке."))
				} else {
					ids, _ := offerService.GetSubscriberIDs()
					for _, tid := range ids {
						bot.Send(tgbotapi.NewMessage(tid, msg.CommandArguments()))
					}
					bot.Send(tgbotapi.NewMessage(chatID, "Рассылка отправлена."))
				}
			}
			continue
		}

		// Ожидаем детали бронирования
		if locID, ok := pendingBooking[userID]; ok {
			details := msg.Text
			delete(pendingBooking, userID)
			user, _ := userRepo.GetByTelegramID(userID)
			if user == nil {
				user, _ = authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
			}
			bookID, err := bookingService.CreateBooking(user.ID, locID, details)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "Ошибка создания брони."))
			} else {
				loc, _ := repository.NewLocationRepository(db).GetByID(locID)
				if loc.ProviderID != nil {
					prov, _ := userRepo.GetByID(*loc.ProviderID)
					textMsg := fmt.Sprintf("Новая бронь от %s по %s: %s", user.FirstName, loc.Name, details)
					btnC := tgbotapi.NewInlineKeyboardButtonData("✔ Подтвердить", fmt.Sprintf("CONFIRM_%d", bookID))
					btnR := tgbotapi.NewInlineKeyboardButtonData("✖ Отклонить", fmt.Sprintf("REJECT_%d", bookID))
					row := tgbotapi.NewInlineKeyboardRow(btnC, btnR)
					notify := tgbotapi.NewMessage(prov.TelegramID, textMsg)
					notify.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)
					bot.Send(notify)
				}
				bot.Send(tgbotapi.NewMessage(chatID, "Заявка отправлена провайдеру."))
			}
			continue
		}

		// Ожидаем фото
		if locID, ok := pendingAddPhoto[userID]; ok {
			if len(msg.Photo) > 0 {
				fileID := msg.Photo[len(msg.Photo)-1].FileID
				locationService.AddPhoto(locID, fileID)
				bot.Send(tgbotapi.NewMessage(chatID, "Фото сохранено."))
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "Ожидается фото."))
			}
			delete(pendingAddPhoto, userID)
			continue
		}

		// Чат турист<->провайдер
		if partnerID := chatService.GetChatPartner(userID); partnerID != 0 {
			out := fmt.Sprintf("%s: %s", msg.From.FirstName, msg.Text)
			bot.Send(tgbotapi.NewMessage(partnerID, out))

			sender, _ := userRepo.GetByTelegramID(userID)
			receiver, _ := userRepo.GetByTelegramID(partnerID)
			bID := chatService.GetChatBookingID(userID)
			messageRepo.Save(&model.Message{FromUserID: sender.ID, ToUserID: receiver.ID, BookingID: &bID, Content: msg.Text, IsSupport: false})
			continue
		}

		// Фоновый поиск локаций
		kw := strings.TrimSpace(text)
		if kw == "*" {
			kw = ""
		}
		locs, err := locationService.SearchLocations("", "", 0, kw)
		if err != nil || len(locs) == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "Ничего не найдено."))
			continue
		}
		btns := make([]tgbotapi.InlineKeyboardButton, len(locs))
		for i, loc := range locs {
			name := loc.Name
			if len(name) > 30 {
				name = name[:30] + "..."
			}
			btns[i] = tgbotapi.NewInlineKeyboardButtonData(name, fmt.Sprintf("LOC_%d", loc.ID))
		}
		rowBtns := tgbotapi.NewInlineKeyboardRow(btns...)
		reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("Найдено: %d", len(locs)))
		reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rowBtns)
		bot.Send(reply)
	}
}
