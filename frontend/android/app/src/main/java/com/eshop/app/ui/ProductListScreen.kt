package com.eshop.app.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Card
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
fun ProductListScreen(onClickProduct: (Long) -> Unit) {
    var list by remember { mutableStateOf<List<Product>>(emptyList()) }
    var message by remember { mutableStateOf("") }

    LaunchedEffect(Unit) {
        runCatching {
            ApiClient.service.listProducts()
        }.onSuccess {
            list = it.list
        }.onFailure {
            message = it.message ?: "load products failed"
        }
    }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        if (message.isNotEmpty()) {
            Text(message)
        }
        LazyColumn {
            items(list) { p ->
                Card(modifier = Modifier
                    .padding(vertical = 6.dp)
                    .clickable { onClickProduct(p.id) }) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Text(text = p.name)
                        Text(text = "¥${p.price_cent / 100.0}  stock:${p.stock}")
                    }
                }
            }
        }
    }
}
