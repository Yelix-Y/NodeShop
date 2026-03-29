package com.eshop.app.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.eshop.app.api.ApiClient
import com.eshop.app.api.Product

@Composable
fun ProductDetailScreen(productId: Long, onAddCart: (Product) -> Unit) {
    var product by remember { mutableStateOf<Product?>(null) }
    var message by remember { mutableStateOf("") }

    LaunchedEffect(productId) {
        runCatching {
            ApiClient.service.getProduct(productId)
        }.onSuccess {
            product = it
        }.onFailure {
            message = it.message ?: "load product failed"
        }
    }

    Column(modifier = Modifier.padding(16.dp)) {
        if (message.isNotEmpty()) Text(message)
        product?.let { p ->
            Text(text = p.name)
            Text(text = p.description)
            Text(text = "¥${p.price_cent / 100.0} stock:${p.stock}")
            Button(
                onClick = { onAddCart(p) },
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp)
            ) {
                Text("Add To Cart")
            }
        }
    }
}
