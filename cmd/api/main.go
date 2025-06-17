package main

import (
	"log"
	"os"
	"path/filepath"

	"tourism/internal/handler"
	"tourism/internal/repository"
	"tourism/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL драйвер
)

func main() {
	// Читаем параметры подключения к БД из переменных окружения
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	dsn := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + " password=" + dbPass + " dbname=" + dbName + " sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	// Выполняем миграции (если есть)
	files, err := filepath.Glob("migrations/*.sql")
	if err == nil {
		for _, file := range files {
			if _, err := db.Exec("BEGIN"); err != nil {
				log.Printf("Ошибка при инициации транзакции миграции: %v", err)
				continue
			}
			err := func() error {
				content, readErr := os.ReadFile(file)
				if readErr != nil {
					return readErr
				}
				if _, execErr := db.Exec(string(content)); execErr != nil {
					return execErr
				}
				return nil
			}()
			if err != nil {
				log.Printf("Миграция %s завершилась ошибкой: %v", file, err)
				db.Exec("ROLLBACK")
			} else {
				db.Exec("COMMIT")
				log.Printf("Миграция %s применена.", file)
			}
		}
	}

	// Инициализируем репозитории
	userRepo := repository.NewUserRepository(db)
	locationRepo := repository.NewLocationRepository(db)
	tripRepo := repository.NewTripRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	// Инициализируем сервисы
	authService := service.NewAuthService(userRepo)
	userService := service.NewUserService(userRepo)
	locationService := service.NewLocationService(locationRepo)
	tripService := service.NewTripService(tripRepo, locationRepo)
	bookingService := service.NewBookingService(bookingRepo)
	chatService := service.NewChatService(bookingRepo, userRepo, locationRepo)
	offerService := service.NewOfferService(subRepo)

	// Создаем Handler и регистрируем маршруты
	h := handler.NewHandler(userService, locationService, tripService, bookingService, chatService, offerService)
	router := gin.Default()
	api := router.Group("/api")
	{
		api.GET("/locations", h.ListLocations)
		api.GET("/users", h.ListUsers)
		// Дополнительные маршруты (например, для добавления локации) могут быть добавлены при необходимости
	}
	// Health-check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Запускаем HTTP-сервер
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
