package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"eshop/internal/repository"
	"eshop/internal/service"
)

type OrderHandler struct {
	service *service.OrderService
}

type createOrderRequest struct {
	ProductID uint64 `json:"product_id" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required,gt=0"`
}

func NewOrderHandler(s *service.OrderService) *OrderHandler {
	return &OrderHandler{service: s}
}

func (h *OrderHandler) Create(c *gin.Context) {
	uid, ok := getCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	idemKey := c.GetHeader("Idempotency-Key")
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	order, err := h.service.CreateOrder(c.Request.Context(), service.CreateOrderInput{
		UserID:    uid,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		IdemKey:   idemKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProductNotFound), errors.Is(err, repository.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, repository.ErrInsufficientStock):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) ListMine(c *gin.Context) {
	uid, ok := getCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := h.service.ListMyOrders(c.Request.Context(), uid, page, size)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total, "page": page, "size": size})
}

func (h *OrderHandler) Pay(c *gin.Context) {
	uid, ok := getCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orderID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	order, err := h.service.PayOrder(c.Request.Context(), uid, orderID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrOrderForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, order)
}

func getCurrentUserID(c *gin.Context) (uint64, bool) {
	uidAny, ok := c.Get(ContextUserID)
	if !ok {
		return 0, false
	}
	uid, ok := uidAny.(uint64)
	return uid, ok
}
