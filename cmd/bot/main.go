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

// клавиатура для туристов
func userKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📍 Найти локации"),
			tgbotapi.NewKeyboardButton("🏨 Бронирование"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🗺 Новый маршрут"),
			tgbotapi.NewKeyboardButton("🛎 Поддержка"),
		),
	)
}

// клавиатура для провайдеров
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

// клавиатура для операторов поддержки
func supportKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📷 Добавить фото"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔍 Проверить локации"),
		),
	)
}

func main() {
	// подключение к БД
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

	// репозитории
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	locRepo := repository.NewLocationRepository(db)
	tripRepo := repository.NewTripRepository(db)
	bookRepo := repository.NewBookingRepository(db)

	// сервисы
	locationService := service.NewLocationService(locRepo)
	tripService := service.NewTripService(tripRepo, locRepo)
	bookingService := service.NewBookingService(bookRepo)
	chatService := service.NewChatService(bookRepo, userRepo, locRepo)
	authService := service.NewAuthService(userRepo)

	// инициализация бота
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Не указан BOT_TOKEN")
	}
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Ошибка инициализации бота: %v", err)
	}
	log.Printf("Бот запущен: @%s", bot.Self.UserName)

	updates := bot.GetUpdatesChan(tgbotapi.NewUpdate(0))

	// временное состояние
	activeTrip := make(map[int64]int)
	pendingBooking := make(map[int64]int)
	pendingAddPhoto := make(map[int64]int)
	searchMode := make(map[int64]string)

	for update := range updates {
		// inline callbacks
		if cq := update.CallbackQuery; cq != nil {
			bot.Request(tgbotapi.NewCallback(cq.ID, ""))
			data := cq.Data
			chatID := cq.Message.Chat.ID
			userID := cq.From.ID

			switch {
			// карточка локации
			case strings.HasPrefix(data, "LOC_"):
				id, _ := strconv.Atoi(strings.TrimPrefix(data, "LOC_"))
				loc, photos, _ := locationService.GetLocationDetails(id)
				if len(photos) > 0 {
					bot.Send(tgbotapi.NewPhoto(chatID, tgbotapi.FileID(photos[0].FileID)))
				}
				// описание + карта
				text := fmt.Sprintf("*%s*\n%s\n[Открыть в картах](https://maps.google.com/?q=%f,%f)",
					loc.Name, loc.Description, loc.Latitude, loc.Longitude,
				)
				msg := tgbotapi.NewMessage(chatID, text)
				msg.ParseMode = "Markdown"
				btnAdd := tgbotapi.NewInlineKeyboardButtonData("➕ В маршрут", fmt.Sprintf("ADDTRIP_%d", id))
				btnBook := tgbotapi.NewInlineKeyboardButtonData("🛎 Забронировать", fmt.Sprintf("BOOK_%d", id))

				switch searchMode[userID] {
				case "location":
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(btnAdd),
					)
				case "booking":
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(btnBook),
					)
				default:
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(btnAdd, btnBook),
					)
				}
				delete(searchMode, userID)
				bot.Send(msg)

			// добавить в маршрут
			case strings.HasPrefix(data, "ADDTRIP_"):
				id, _ := strconv.Atoi(strings.TrimPrefix(data, "ADDTRIP_"))
				if tripID, ok := activeTrip[userID]; ok {
					tripService.AddLocationToTrip(tripID, id)
					bot.Send(tgbotapi.NewMessage(chatID, "Локация добавлена в маршрут"))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Сначала создайте маршрут (🗺 Новый маршрут)"))
				}

			// запрос деталей брони по локации
			case strings.HasPrefix(data, "BOOK_"):
				id, _ := strconv.Atoi(strings.TrimPrefix(data, "BOOK_"))
				pendingBooking[userID] = id
				bot.Send(tgbotapi.NewMessage(chatID,
					"Укажите даты и количество участников, напр.: 2025-07-01 — 2025-07-05, 3 человека",
				))

			// подтверждение/отказ провайдером
			case strings.HasPrefix(data, "CONFIRM_"), strings.HasPrefix(data, "REJECT_"):
				parts := strings.Split(data, "_")
				action, sid := parts[0], parts[1]
				bID, _ := strconv.Atoi(sid)
				if action == "CONFIRM" {
					bookingService.ConfirmBooking(bID)
				} else {
					bookingService.RejectBooking(bID)
				}
				bk, _ := bookingService.GetBooking(bID)
				res := map[string]string{
					"CONFIRM": "Ваша бронь подтверждена ✅",
					"REJECT":  "Ваша бронь отклонена ❌",
				}[action]
				bot.Send(tgbotapi.NewMessage(int64(bk.UserID), res))
			}

			continue
		}

		// обработка сообщений
		if msg := update.Message; msg != nil {
			chatID := msg.Chat.ID
			userID := msg.From.ID
			text := msg.Text

			// детали брони
			if locID, ok := pendingBooking[userID]; ok {
				bookID, err := bookingService.CreateBooking(int(userID), locID, text)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка создания брони"))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID,
						fmt.Sprintf("Заявка #%d отправлена провайдеру", bookID),
					))
					loc, _ := locRepo.GetByID(locID)
					if loc.ProviderID != nil {
						prov, _ := userRepo.GetByID(*loc.ProviderID)
						notify := tgbotapi.NewMessage(prov.TelegramID,
							fmt.Sprintf("Новая бронь #%d от %s: %s", bookID, msg.From.FirstName, text),
						)
						btnC := tgbotapi.NewInlineKeyboardButtonData("✔ Подтвердить", fmt.Sprintf("CONFIRM_%d", bookID))
						btnR := tgbotapi.NewInlineKeyboardButtonData("✖ Отклонить", fmt.Sprintf("REJECT_%d", bookID))
						notify.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(btnC, btnR),
						)
						bot.Send(notify)
					}
				}
				delete(pendingBooking, userID)
				continue
			}

			// добавление фото
			if locID, ok := pendingAddPhoto[userID]; ok {
				if len(msg.Photo) > 0 {
					fileID := msg.Photo[len(msg.Photo)-1].FileID
					locationService.AddPhoto(locID, fileID)
					bot.Send(tgbotapi.NewMessage(chatID, "Фото сохранено"))
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Ожидается фото"))
				}
				delete(pendingAddPhoto, userID)
				continue
			}

			// чат турист ↔ провайдер
			if partner := chatService.GetChatPartner(userID); partner != 0 {
				out := fmt.Sprintf("%s: %s", msg.From.FirstName, text)
				bot.Send(tgbotapi.NewMessage(partner, out))
				// логирование
				s, _ := userRepo.GetByTelegramID(userID)
				r, _ := userRepo.GetByTelegramID(partner)
				bID := chatService.GetChatBookingID(userID)
				messageRepo.Save(&model.Message{
					FromUserID: s.ID,
					ToUserID:   r.ID,
					BookingID:  &bID,
					Content:    text,
					IsSupport:  false,
				})
				continue
			}

			// команда /start
			if msg.IsCommand() && msg.Command() == "start" {
				user, _ := authService.AuthUser(int64(userID), msg.From.UserName, msg.From.FirstName, msg.From.LastName)
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

			// кнопки меню
			switch text {
			case "📍 Найти локации":
				searchMode[userID] = "location"
				bot.Send(tgbotapi.NewMessage(chatID, "Введите слово для поиска (или * для всех):"))
				continue

			case "🏨 Бронирование":
				searchMode[userID] = "booking"
				bot.Send(tgbotapi.NewMessage(chatID, "Введите слово для поиска (или * для всех):"))
				continue

			case "🗺 Новый маршрут":
				user, _ := userRepo.GetByTelegramID(userID)
				if user == nil {
					user, _ = authService.AuthUser(int64(userID), msg.From.UserName, msg.From.FirstName, msg.From.LastName)
				}
				tripID, err := tripService.CreateTrip(user.ID, "Мой маршрут")
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Не удалось создать маршрут."))
				} else {
					activeTrip[userID] = tripID
					bot.Send(tgbotapi.NewMessage(chatID, "Маршрут создан. Добавляйте локации."))
				}
				continue

			case "🛎 Поддержка":
				bot.Send(tgbotapi.NewMessage(chatID, "Перейдите в @TouristSupportHelpBot для поддержки."))
				continue

			case "📦 Мои бронирования":
				bot.Send(tgbotapi.NewMessage(chatID, "Функция пока не реализована."))
				continue

			case "📷 Добавить фото":
				bot.Send(tgbotapi.NewMessage(chatID, "Используйте /addphoto <ID_локации> чтобы добавить фото."))
				continue

			case "🔍 Проверить локации":
				bot.Send(tgbotapi.NewMessage(chatID, "Функция пока не реализована."))
				continue
			}

			// команды
			if msg.IsCommand() {
				switch msg.Command() {
				case "locations":
					bot.Send(tgbotapi.NewMessage(chatID, "Введите слово для поиска (или * для всех):"))
				case "newtrip":
					user, _ := userRepo.GetByTelegramID(userID)
					if user == nil {
						user, _ = authService.AuthUser(int64(userID), msg.From.UserName, msg.From.FirstName, msg.From.LastName)
					}
					tripID, err := tripService.CreateTrip(user.ID, "Мой маршрут")
					if err != nil {
						bot.Send(tgbotapi.NewMessage(chatID, "Не удалось создать маршрут."))
					} else {
						activeTrip[userID] = tripID
						bot.Send(tgbotapi.NewMessage(chatID, "Маршрут создан. Добавляйте локации."))
					}
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
				}
				continue
			}

			// фоновый поиск локаций по тексту
			_, hasMode := searchMode[userID]
			kw := strings.TrimSpace(text)
			if kw == "*" {
				kw = ""
			}
			locs, err := locationService.SearchLocations("", "", 0, kw)
			if err != nil || len(locs) == 0 {
				bot.Send(tgbotapi.NewMessage(chatID, "Ничего не найдено."))
				if hasMode {
					delete(searchMode, userID)
				}
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
}
