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
	// Подключение к базе данных (аналогично API)
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

	// Инициализация репозиториев и сервисов
	userRepo := repository.NewUserRepository(db)
	locationRepo := repository.NewLocationRepository(db)
	tripRepo := repository.NewTripRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)

	authService := service.NewAuthService(userRepo)
	userService := service.NewUserService(userRepo)
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

	// Карты для отслеживания состояния диалога
	activeTrip := make(map[int64]int)      // текущий активный маршрут (TripID) для пользователя
	pendingBooking := make(map[int64]int)  // ожидаем детали бронирования для указанной локации
	pendingAddPhoto := make(map[int64]int) // ожидаем отправки фото для указанной локации (администратор)

	for update := range updates {
		// 1. Обработка нажатий инлайн-кнопок (CallbackQuery)
		if update.CallbackQuery != nil {
			callbackID := update.CallbackQuery.ID
			bot.Request(tgbotapi.NewCallback(callbackID, ""))
			data := update.CallbackQuery.Data
			fromID := update.CallbackQuery.From.ID

			if strings.HasPrefix(data, "LOC_") {
				// Выбрана локация из списка
				locIDStr := strings.TrimPrefix(data, "LOC_")
				locID, _ := strconv.Atoi(locIDStr)
				location, photos, err := locationService.GetLocationDetails(locID)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(fromID, "Ошибка при получении данных о локации."))
				} else {
					// Отправляем фотографии локации
					if len(photos) > 0 {
						for _, ph := range photos {
							photoMsg := tgbotapi.NewPhoto(fromID, tgbotapi.FileID(ph.FileID))
							bot.Send(photoMsg)
						}
					}
					// Описание и ссылка
					text := fmt.Sprintf("*%s*\n%s\n\n[Открыть в картах](https://maps.google.com/?q=%f,%f)",
						location.Name, location.Description, location.Latitude, location.Longitude)
					msg := tgbotapi.NewMessage(fromID, text)
					msg.ParseMode = "Markdown"
					buttons := []tgbotapi.InlineKeyboardButton{
						tgbotapi.NewInlineKeyboardButtonData("Добавить в маршрут", fmt.Sprintf("ADDTRIP_%d", location.ID)),
						tgbotapi.NewInlineKeyboardButtonData("Забронировать", fmt.Sprintf("BOOK_%d", location.ID)),
					}
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
					bot.Send(msg)
				}
			} else if strings.HasPrefix(data, "ADDTRIP_") {
				locIDStr := strings.TrimPrefix(data, "ADDTRIP_")
				locID, _ := strconv.Atoi(locIDStr)
				tripID, ok := activeTrip[fromID]
				reply := ""
				if !ok {
					reply = "У вас нет активной поездки. Введите /newtrip для создания маршрута."
				} else {
					err := tripService.AddLocationToTrip(tripID, locID)
					if err != nil {
						reply = "Ошибка добавления локации в маршрут."
					} else {
						reply = "Локация добавлена в ваш маршрут."
					}
				}
				bot.Send(tgbotapi.NewMessage(fromID, reply))
			} else if strings.HasPrefix(data, "BOOK_") {
				locIDStr := strings.TrimPrefix(data, "BOOK_")
				locID, _ := strconv.Atoi(locIDStr)
				pendingBooking[fromID] = locID
				bot.Send(tgbotapi.NewMessage(fromID, "Пожалуйста, отправьте детали бронирования (даты, количество людей и др.)."))
			} else if strings.HasPrefix(data, "CONFIRM_") {
				bookIDStr := strings.TrimPrefix(data, "CONFIRM_")
				bookID, _ := strconv.Atoi(bookIDStr)
				bookingService.ConfirmBooking(bookID)
				booking, _ := bookingService.GetBooking(bookID)
				touristUser, _ := userRepo.GetByID(booking.UserID)
				bot.Send(tgbotapi.NewMessage(touristUser.TelegramID, "Ваше бронирование подтверждено!"))
				bot.Send(tgbotapi.NewMessage(fromID, "Вы подтвердили бронирование."))
			} else if strings.HasPrefix(data, "REJECT_") {
				bookIDStr := strings.TrimPrefix(data, "REJECT_")
				bookID, _ := strconv.Atoi(bookIDStr)
				bookingService.RejectBooking(bookID)
				booking, _ := bookingService.GetBooking(bookID)
				touristUser, _ := userRepo.GetByID(booking.UserID)
				bot.Send(tgbotapi.NewMessage(touristUser.TelegramID, "Ваше бронирование отклонено."))
				bot.Send(tgbotapi.NewMessage(fromID, "Вы отклонили бронирование."))
			}
			continue
		}

		// 2. Обработка текстовых сообщений
		if update.Message == nil {
			continue
		}
		msg := update.Message
		chatID := msg.Chat.ID
		userTelegramID := msg.From.ID

		if msg.IsCommand() {
			switch msg.Command() {
			case "start":
				user, err := authService.AuthUser(userTelegramID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка авторизации."))
				} else {
					welcome := fmt.Sprintf("Здравствуйте, %s! Добро пожаловать в Tourist Support.", user.FirstName)
					bot.Send(tgbotapi.NewMessage(chatID, welcome))
				}
			case "locations":
				bot.Send(tgbotapi.NewMessage(chatID, "Введите ключевое слово для поиска локаций (или отправьте '*' для просмотра всех):"))
			case "newtrip":
				user, err := userRepo.GetByTelegramID(userTelegramID)
				if err != nil {
					user, _ = authService.AuthUser(userTelegramID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				tripID, err := tripService.CreateTrip(user.ID, "Мой маршрут")
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании нового маршрута."))
				} else {
					activeTrip[userTelegramID] = tripID
					bot.Send(tgbotapi.NewMessage(chatID, "Новый маршрут создан. Теперь вы можете добавлять локации командой \"Добавить в маршрут\" в описании локации."))
				}
			case "support":
				bot.Send(tgbotapi.NewMessage(chatID, "Для связи с оператором поддержки перейдите в чат со службой поддержки @TouristSupportHelpBot (бот поддержки)."))
			case "subscribe_offers":
				user, err := userRepo.GetByTelegramID(userTelegramID)
				if err != nil {
					user, _ = authService.AuthUser(userTelegramID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				offerService.Subscribe(user.ID)
				bot.Send(tgbotapi.NewMessage(chatID, "Вы подписаны на рассылку интересных предложений."))
			case "unsubscribe_offers":
				user, err := userRepo.GetByTelegramID(userTelegramID)
				if err != nil {
					user, _ = authService.AuthUser(userTelegramID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				offerService.Unsubscribe(user.ID)
				bot.Send(tgbotapi.NewMessage(chatID, "Вы отписаны от рассылки предложений."))
			case "addphoto":
				args := msg.CommandArguments()
				if args == "" {
					bot.Send(tgbotapi.NewMessage(chatID, "Использование: /addphoto <ID_локации>"))
				} else {
					locID, err := strconv.Atoi(args)
					if err != nil {
						bot.Send(tgbotapi.NewMessage(chatID, "Некорректный ID локации."))
					} else {
						user, err := userRepo.GetByTelegramID(userTelegramID)
						if err != nil {
							bot.Send(tgbotapi.NewMessage(chatID, "Вы не зарегистрированы."))
						} else if user.Role != "support" {
							bot.Send(tgbotapi.NewMessage(chatID, "У вас нет прав для добавления фото."))
						} else {
							pendingAddPhoto[userTelegramID] = locID
							bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Отправьте фотографию для локации #%d", locID)))
						}
					}
				}
			case "broadcast":
				user, err := userRepo.GetByTelegramID(userTelegramID)
				if err != nil || user.Role != "support" {
					bot.Send(tgbotapi.NewMessage(chatID, "Команда недоступна."))
				} else {
					msgText := msg.CommandArguments()
					if msgText == "" {
						bot.Send(tgbotapi.NewMessage(chatID, "Использование: /broadcast <сообщение>"))
					} else {
						ids, err := offerService.GetSubscriberIDs()
						if err != nil {
							bot.Send(tgbotapi.NewMessage(chatID, "Не удалось получить список подписчиков."))
						} else {
							for _, tid := range ids {
								bot.Send(tgbotapi.NewMessage(tid, msgText))
							}
							bot.Send(tgbotapi.NewMessage(chatID, "Рассылка отправлена подписчикам."))
						}
					}
				}
			}
			continue
		}

		// Если ожидаются детали бронирования
		if locID, ok := pendingBooking[userTelegramID]; ok {
			details := msg.Text
			delete(pendingBooking, userTelegramID)
			user, _ := userRepo.GetByTelegramID(userTelegramID)
			if user == nil {
				user, _ = authService.AuthUser(userTelegramID, msg.From.UserName, msg.From.FirstName, msg.From.LastName)
			}
			bookingID, err := bookingService.CreateBooking(user.ID, locID, details)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "Не удалось создать бронирование."))
			} else {
				location, _ := locationRepo.GetByID(locID)
				if location.ProviderID != nil {
					provider, err := userRepo.GetByID(*location.ProviderID)
					if err == nil {
						text := fmt.Sprintf("Новая заявка на бронирование от пользователя %s по локации \"%s\".\nДетали: %s", msg.From.FirstName, location.Name, details)
						confirmBtn := tgbotapi.NewInlineKeyboardButtonData("Подтвердить", fmt.Sprintf("CONFIRM_%d", bookingID))
						rejectBtn := tgbotapi.NewInlineKeyboardButtonData("Отклонить", fmt.Sprintf("REJECT_%d", bookingID))
						markup := tgbotapi.NewInlineKeyboardMarkup(confirmBtn, rejectBtn)
						notify := tgbotapi.NewMessage(provider.TelegramID, text)
						notify.ReplyMarkup = markup
						bot.Send(notify)
					}
				}
				bot.Send(tgbotapi.NewMessage(chatID, "Ваша заявка отправлена. Ожидайте подтверждения от провайдера."))
			}
			continue
		}
		// Если ожидается фото для локации
		if locID, ok := pendingAddPhoto[userTelegramID]; ok {
			if msg.Photo != nil && len(*msg.Photo) > 0 {
				photos := *msg.Photo
				fileID := photos[len(photos)-1].FileID
				err := locationService.AddPhoto(locID, fileID)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения фото."))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Фото добавлено в галерею локации."))
				}
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "Ожидается фотография. Пожалуйста, отправьте изображение."))
			}
			delete(pendingAddPhoto, userTelegramID)
			continue
		}
		// Если пользователь в режиме чата турист↔поставщик
		partnerID := chatService.GetChatPartner(userTelegramID)
		if partnerID != 0 {
			partner, err := userRepo.GetByTelegramID(partnerID)
			if err == nil {
				senderName := msg.From.FirstName
				outText := fmt.Sprintf("%s: %s", senderName, msg.Text)
				bot.Send(tgbotapi.NewMessage(partnerID, outText))
				sender, _ := userRepo.GetByTelegramID(userTelegramID)
				receiver, _ := userRepo.GetByTelegramID(partnerID)
				bookingID := chatService.GetChatBookingID(userTelegramID)
				messageRepo.Save(&model.Message{FromUserID: sender.ID, ToUserID: receiver.ID, BookingID: &bookingID, Content: msg.Text, IsSupport: false})
			}
			continue
		}
		// Иначе обрабатываем сообщение как поисковый запрос по локациям
		query := strings.TrimSpace(msg.Text)
		var keyword string
		if query == "*" || strings.ToLower(query) == "все" {
			keyword = ""
		} else {
			keyword = query
		}
		locations, err := locationService.SearchLocations("", "", 0, keyword)
		if err != nil || len(locations) == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "По вашему запросу ничего не найдено."))
		} else {
			var buttons []tgbotapi.InlineKeyboardButton
			for _, loc := range locations {
				btnText := loc.Name
				if len(btnText) > 30 {
					btnText = btnText[:30] + "..."
				}
				buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("LOC_%d", loc.ID)))
			}
			replyText := fmt.Sprintf("Найдено результатов: %d", len(locations))
			msgReply := tgbotapi.NewMessage(chatID, replyText)
			if len(buttons) > 0 {
				msgReply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
			}
			bot.Send(msgReply)
		}
	}
}
