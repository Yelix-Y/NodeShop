package com.eshop.app.api

import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

interface ApiService {
    @POST("/api/v1/users/register")
    suspend fun register(@Body req: RegisterRequest)

    @POST("/api/v1/users/login")
    suspend fun login(@Body req: LoginRequest): LoginResponse

    @GET("/api/v1/products")
    suspend fun listProducts(
        @Query("page") page: Int = 1,
        @Query("page_size") pageSize: Int = 20
    ): ProductListResponse

    @GET("/api/v1/products/{id}")
    suspend fun getProduct(@Path("id") id: Long): Product

    @GET("/api/v1/orders/my")
    suspend fun listMyOrders(
        @Header("Authorization") authorization: String,
        @Query("page") page: Int = 1,
        @Query("page_size") pageSize: Int = 20
    ): OrderListResponse

    @POST("/api/v1/orders")
    suspend fun createOrder(
        @Header("Authorization") authorization: String,
        @Header("Idempotency-Key") idemKey: String,
        @Body req: CreateOrderRequest
    )
}
