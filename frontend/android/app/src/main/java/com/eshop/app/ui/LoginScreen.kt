package com.eshop.app.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedTextField
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
import com.eshop.app.api.LoginRequest
import com.eshop.app.api.RegisterRequest
import kotlinx.coroutines.launch

@Composable
fun LoginScreen(onLoginSuccess: (String) -> Unit) {
    var username by remember { mutableStateOf("alice") }
    var password by remember { mutableStateOf("123456") }
    var nickname by remember { mutableStateOf("Alice") }
    var message by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    Column(modifier = Modifier.padding(16.dp)) {
        OutlinedTextField(
            value = username,
            onValueChange = { username = it },
            label = { Text("Username") },
            modifier = Modifier.fillMaxWidth()
        )
        OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            label = { Text("Password") },
            modifier = Modifier.fillMaxWidth()
        )
        OutlinedTextField(
            value = nickname,
            onValueChange = { nickname = it },
            label = { Text("Nickname") },
            modifier = Modifier.fillMaxWidth()
        )

        Button(onClick = {
            scope.launch {
                runCatching {
                    ApiClient.service.register(RegisterRequest(username, password, nickname))
                }.onSuccess {
                    message = "register success"
                }.onFailure {
                    message = it.message ?: "register failed"
                }
            }
        }) {
            Text("Register")
        }

        Button(onClick = {
            scope.launch {
                runCatching {
                    ApiClient.service.login(LoginRequest(username, password))
                }.onSuccess {
                    onLoginSuccess(it.token)
                }.onFailure {
                    message = it.message ?: "login failed"
                }
            }
        }) {
            Text("Login")
        }

        Text(text = message, modifier = Modifier.padding(top = 8.dp))
    }
}
