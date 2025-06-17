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

func main() {
	// Подключение к базе данных
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

	// Инициализация репозиториев и сервисов
	userRepo := repository.NewUserRepository(db)
	locationRepo := repository.NewLocationRepository(db)
	tripRepo := repository.NewTripRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)

	authService := service.NewAuthService(userRepo)
	locationService := service.NewLocationService(locationRepo)
	tripService := service.NewTripService(tripRepo, locationRepo)
	bookingService := service.NewBookingService(bookingRepo)
	chatService := service.NewChatService(bookingRepo, userRepo, locationRepo)
	offerService := service.NewOfferService(subRepo)

	// Инициализация Telegram Bot API
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

	// Состояние диалогов
	activeTrip := make(map[int64]int)      // userID -> TripID
	pendingBooking := make(map[int64]int)  // userID -> LocationID
	pendingAddPhoto := make(map[int64]int) // userID -> LocationID

	for update := range updates {
		// --- CallbackQuery (inline buttons) ---
		if cq := update.CallbackQuery; cq != nil {
			bot.Request(tgbotapi.NewCallback(cq.ID, ""))

			fromID := cq.From.ID
			data := cq.Data

			switch {
			// Показ деталей локации
			case strings.HasPrefix(data, "LOC_"):
				locID, _ := strconv.Atoi(strings.TrimPrefix(data, "LOC_"))
				loc, photos, err := locationService.GetLocationDetails(locID)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(fromID, "Ошибка получения локации."))
					continue
				}

				// Галерея фото
				for _, ph := range photos {
					bot.Send(tgbotapi.NewPhoto(fromID, tgbotapi.FileID(ph.FileID)))
				}

				// Описание + карта
				text := fmt.Sprintf(
					"*%s*\n%s\n\n[Открыть в картах](https://maps.google.com/?q=%f,%f)",
					loc.Name, loc.Description, loc.Latitude, loc.Longitude,
				)
				msg := tgbotapi.NewMessage(fromID, text)
				msg.ParseMode = "Markdown"

				// Кнопки
				btnAdd := tgbotapi.NewInlineKeyboardButtonData("Добавить в маршрут", fmt.Sprintf("ADDTRIP_%d", loc.ID))
				btnBook := tgbotapi.NewInlineKeyboardButtonData("Забронировать", fmt.Sprintf("BOOK_%d", loc.ID))
				row := tgbotapi.NewInlineKeyboardRow(btnAdd, btnBook)
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)

				bot.Send(msg)

			// Добавление в маршрут
			case strings.HasPrefix(data, "ADDTRIP_"):
				locID, _ := strconv.Atoi(strings.TrimPrefix(data, "ADDTRIP_"))
				if tripID, ok := activeTrip[fromID]; ok {
					if err := tripService.AddLocationToTrip(tripID, locID); err != nil {
						bot.Send(tgbotapi.NewMessage(fromID, "Ошибка добавления локации."))
					} else {
						bot.Send(tgbotapi.NewMessage(fromID, "Локация добавлена в маршрут."))
					}
				} else {
					bot.Send(tgbotapi.NewMessage(fromID, "Нет активного маршрута. Введите /newtrip."))
				}

			// Начать бронирование
			case strings.HasPrefix(data, "BOOK_"):
				locID, _ := strconv.Atoi(strings.TrimPrefix(data, "BOOK_"))
				pendingBooking[fromID] = locID
				bot.Send(tgbotapi.NewMessage(fromID, "Отправьте детали брони (даты, число людей)."))

			// Подтвердить бронирование (для провайдера)
			case strings.HasPrefix(data, "CONFIRM_"):
				bookID, _ := strconv.Atoi(strings.TrimPrefix(data, "CONFIRM_"))
				bookingService.ConfirmBooking(bookID)
				booking, _ := bookingService.GetBooking(bookID)
				tourist, _ := userRepo.GetByID(booking.UserID)
				bot.Send(tgbotapi.NewMessage(tourist.TelegramID, "Ваше бронирование подтверждено!"))
				bot.Send(tgbotapi.NewMessage(fromID, "Вы подтвердили бронирование."))

			// Отклонить бронирование
			case strings.HasPrefix(data, "REJECT_"):
				bookID, _ := strconv.Atoi(strings.TrimPrefix(data, "REJECT_"))
				bookingService.RejectBooking(bookID)
				booking, _ := bookingService.GetBooking(bookID)
				tourist, _ := userRepo.GetByID(booking.UserID)
				bot.Send(tgbotapi.NewMessage(tourist.TelegramID, "Ваше бронирование отклонено."))
				bot.Send(tgbotapi.NewMessage(fromID, "Вы отклонили бронирование."))
			}

			continue
		}

		// --- Обычные сообщения ---
		if update.Message == nil {
			continue
		}
		msg := update.Message
		chatID := msg.Chat.ID
		userID := msg.From.ID

		// Команды
		if msg.IsCommand() {
			switch msg.Command() {
			case "start":
				user, err := authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка авторизации."))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Здравствуйте, %s!", user.FirstName)))
				}

			case "locations":
				bot.Send(tgbotapi.NewMessage(chatID, "Введите слово для поиска (или * для всех):"))

			case "newtrip":
				user, err := userRepo.GetByTelegramID(userID)
				if err != nil {
					user, _ = authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				tripID, err := tripService.CreateTrip(user.ID, "Мой маршрут")
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Не удалось создать маршрут."))
				} else {
					activeTrip[userID] = tripID
					bot.Send(tgbotapi.NewMessage(chatID, "Маршрут создан. Добавляйте локации."))
				}

			case "support":
				bot.Send(tgbotapi.NewMessage(chatID, "Перейдите в @TouristSupportHelpBot для поддержки."))

			case "subscribe_offers":
				user, err := userRepo.GetByTelegramID(userID)
				if err != nil {
					user, _ = authService.AuthUser(userID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				offerService.Subscribe(user.ID)
				bot.Send(tgbotapi.NewMessage(chatID, "Вы подписаны на рассылку предложений."))

			case "unsubscribe_offers":
				user, err := userRepo.GetByTelegramID(userID)
				if err != nil {
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
					text := msg.CommandArguments()
					if text == "" {
						bot.Send(tgbotapi.NewMessage(chatID, "Используйте: /broadcast <текст>"))
					} else {
						ids, _ := offerService.GetSubscriberIDs()
						for _, tid := range ids {
							bot.Send(tgbotapi.NewMessage(tid, text))
						}
						bot.Send(tgbotapi.NewMessage(chatID, "Рассылка отправлена."))
					}
				}
			}
			continue
		}

		// Обработка «ожидающих» состояний

		// Детали бронирования
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
				loc, _ := locationRepo.GetByID(locID)
				if loc.ProviderID != nil {
					prov, _ := userRepo.GetByID(*loc.ProviderID)
					text := fmt.Sprintf(
						"Новая бронь: %s ▶ %s\n%s",
						user.FirstName, loc.Name, details,
					)
					btnC := tgbotapi.NewInlineKeyboardButtonData("✔ Подтвердить", fmt.Sprintf("CONFIRM_%d", bookID))
					btnR := tgbotapi.NewInlineKeyboardButtonData("✖ Отклонить", fmt.Sprintf("REJECT_%d", bookID))
					row := tgbotapi.NewInlineKeyboardRow(btnC, btnR)
					notify := tgbotapi.NewMessage(prov.TelegramID, text)
					notify.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)
					bot.Send(notify)
				}
				bot.Send(tgbotapi.NewMessage(chatID, "Заявка отправлена провайдеру."))
			}
			continue
		}

		// Добавление фото
		if locID, ok := pendingAddPhoto[userID]; ok {
			if len(msg.Photo) > 0 {
				fileID := msg.Photo[len(msg.Photo)-1].FileID
				err := locationService.AddPhoto(locID, fileID)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения фото."))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Фото добавлено."))
				}
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "Ожидается фото."))
			}
			delete(pendingAddPhoto, userID)
			continue
		}

		// Режим чат турист↔провайдер
		if partnerID := chatService.GetChatPartner(userID); partnerID != 0 {
			out := fmt.Sprintf("%s: %s", msg.From.FirstName, msg.Text)
			bot.Send(tgbotapi.NewMessage(partnerID, out))

			// Сохраняем в БД
			sender, _ := userRepo.GetByTelegramID(userID)
			receiver, _ := userRepo.GetByTelegramID(partnerID)
			bID := chatService.GetChatBookingID(userID)
			messageRepo.Save(&model.Message{
				FromUserID: sender.ID,
				ToUserID:   receiver.ID,
				BookingID:  &bID,
				Content:    msg.Text,
				IsSupport:  false,
			})
			continue
		}

		// Поиск локаций по тексту
		kw := strings.TrimSpace(msg.Text)
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
		row := tgbotapi.NewInlineKeyboardRow(btns...)
		reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("Найдено: %d", len(locs)))
		reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)
		bot.Send(reply)
	}
}
