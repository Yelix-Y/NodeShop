package com.eshop.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import com.eshop.app.api.Product
import com.eshop.app.data.SessionStore
import com.eshop.app.ui.CartItem
import com.eshop.app.ui.CartScreen
import com.eshop.app.ui.CheckoutScreen
import com.eshop.app.ui.LoginScreen
import com.eshop.app.ui.ProductDetailScreen
import com.eshop.app.ui.ProductListScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            var page by remember { mutableStateOf("login") }
            var selectedProductId by remember { mutableStateOf(0L) }
            val cart = remember { mutableStateListOf<CartItem>() }

            when (page) {
                "login" -> LoginScreen(onLoginSuccess = {
                    SessionStore.token = it
                    page = "products"
                })

                "products" -> ProductListScreen(onClickProduct = {
                    selectedProductId = it
                    page = "detail"
                })

                "detail" -> ProductDetailScreen(productId = selectedProductId, onAddCart = { p: Product ->
                    val old = cart.firstOrNull { it.product.id == p.id }
                    if (old == null) {
                        cart.add(CartItem(product = p, quantity = 1))
                    } else {
                        old.quantity += 1
                    }
                    page = "cart"
                })

                "cart" -> CartScreen(items = cart, onCheckout = {
                    page = "checkout"
                })

                "checkout" -> CheckoutScreen(token = SessionStore.token, cart = cart, onDone = {
                    cart.clear()
                    page = "products"
                })

                else -> Text("unknown page")
            }
        }
    }
}
