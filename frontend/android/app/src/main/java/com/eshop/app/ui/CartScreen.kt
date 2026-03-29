package com.eshop.app.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.eshop.app.api.Product

data class CartItem(
    val product: Product,
    var quantity: Long
)

@Composable
fun CartScreen(items: List<CartItem>, onCheckout: () -> Unit) {
    Column(modifier = Modifier.padding(16.dp)) {
        LazyColumn(modifier = Modifier.weight(1f, fill = false)) {
            items(items) { item ->
                Text(text = "${item.product.name} x ${item.quantity}")
                Text(text = "¥${(item.product.price_cent * item.quantity) / 100.0}", modifier = Modifier.padding(bottom = 8.dp))
            }
        }
        Button(onClick = onCheckout, modifier = Modifier.fillMaxWidth()) {
            Text("Checkout")
        }
    }
}
