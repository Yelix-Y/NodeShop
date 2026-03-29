package com.eshop.app.api

data class RegisterRequest(
    val username: String,
    val password: String,
    val nickname: String
)

data class LoginRequest(
    val username: String,
    val password: String
)

data class LoginResponse(
    val token: String
)

data class Product(
    val id: Long,
    val sku: String?,
    val name: String,
    val description: String,
    val price_cent: Long,
    val stock: Long,
    val status: Int
)

data class ProductListResponse(
    val list: List<Product>,
    val total: Long,
    val page: Int,
    val size: Int
)

data class CreateOrderRequest(
    val product_id: Long,
    val quantity: Long
)

data class OrderItem(
    val id: Long,
    val order_no: String,
    val user_id: Long,
    val product_id: Long,
    val quantity: Long,
    val total_price_cent: Long,
    val status: String
)

data class OrderListResponse(
    val list: List<OrderItem>,
    val total: Long,
    val page: Int,
    val size: Int
)
