package com.eshop.app.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.eshop.app.api.ApiClient
import com.eshop.app.api.CreateOrderRequest
import kotlinx.coroutines.launch

@Composable
fun CheckoutScreen(
    token: String,
    cart: List<CartItem>,
    onDone: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var message by remember { mutableStateOf("") }

    Column(modifier = Modifier.padding(16.dp)) {
        Button(
            onClick = {
                scope.launch {
                    runCatching {
                        cart.forEach { item ->
                            ApiClient.service.createOrder(
                                authorization = "Bearer $token",
                                idemKey = "android-order-${item.product.id}-${System.currentTimeMillis()}",
                                req = CreateOrderRequest(item.product.id, item.quantity)
                            )
                        }
                    }.onSuccess {
                        message = "order created"
                        onDone()
                    }.onFailure {
                        message = it.message ?: "checkout failed"
                    }
                }
            },
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Submit Order")
        }
        Text(text = message, modifier = Modifier.padding(top = 8.dp))
    }
}
