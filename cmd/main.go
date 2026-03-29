package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	v1 "eshop/api/v1"
	"eshop/internal/repository"
	"eshop/internal/service"
)

func main() {
	ctx := context.Background()

	db, err := initMySQL()
	if err != nil {
		log.Fatalf("init mysql failed: %v", err)
	}

	redisClient, err := initRedis(ctx)
	if err != nil {
		log.Fatalf("init redis failed: %v", err)
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Printf("close redis failed: %v", closeErr)
		}
	}()

	userRepo := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db, redisClient)
	orderRepo := repository.NewOrderRepository(db)

	jwtSecret := getEnv("JWT_SECRET", "dev-secret-change-me")
	userService := service.NewUserService(userRepo, jwtSecret)
	productService := service.NewProductService(productRepo)
	orderService := service.NewOrderService(orderRepo, productRepo)

	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())
	v1.RegisterRoutes(router, v1.RouterDependencies{
		UserService:    userService,
		ProductService: productService,
		OrderService:   orderService,
		JWTSecret:      jwtSecret,
	})

	server := &http.Server{
		Addr:              getEnv("HTTP_ADDR", ":8080"),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server listen failed: %v", err)
		}
	}()

	log.Printf("eshop server started at %s", server.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("server exited")
}

func initMySQL() (*gorm.DB, error) {
	dsn := getEnv("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/eshop?charset=utf8mb4&parseTime=True&loc=Local")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get db instance: %w", err)
	}
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

func initRedis(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           0,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		PoolSize:     100,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
