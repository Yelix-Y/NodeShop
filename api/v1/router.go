package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"eshop/internal/service"
	"eshop/internal/utils"
)

const ContextUserID = "current_user_id"

type RouterDependencies struct {
	UserService    *service.UserService
	ProductService *service.ProductService
	OrderService   *service.OrderService
	JWTSecret      string
}

func RegisterRoutes(r *gin.Engine, deps RouterDependencies) {
	userHandler := NewUserHandler(deps.UserService)
	productHandler := NewProductHandler(deps.ProductService)
	orderHandler := NewOrderHandler(deps.OrderService)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "eshop", "status": "ok"})
	})

	api := r.Group("/api/v1")
	users := api.Group("/users")
	{
		users.POST("/register", userHandler.Register)
		users.POST("/login", userHandler.Login)
		users.GET("/me", AuthMiddleware(deps.JWTSecret), userHandler.Me)
	}

	products := api.Group("/products")
	{
		products.GET("", productHandler.List)
		products.GET("/:id", productHandler.Get)
		products.POST("", productHandler.Create)
		products.PUT("/:id", productHandler.Update)
	}

	orders := api.Group("/orders")
	orders.Use(AuthMiddleware(deps.JWTSecret))
	{
		orders.POST("", orderHandler.Create)
		orders.GET("/my", orderHandler.ListMine)
		orders.POST("/:id/pay", orderHandler.Pay)
	}
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ParseJWT(jwtSecret, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Next()
	}
}
