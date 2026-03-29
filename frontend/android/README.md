# Android Demo (Kotlin)

这是前端 Android 示例代码目录（Kotlin + Jetpack Compose + Retrofit）。

## 目录

- `api/`: Retrofit 接口和数据模型
- `data/`: 简单会话存储
- `ui/`: 登录、商品、购物车、结算页面示例
- `MainActivity.kt`: 页面流转示例

## 依赖建议（`app/build.gradle.kts`）

```kotlin
implementation("com.squareup.retrofit2:retrofit:2.11.0")
implementation("com.squareup.retrofit2:converter-gson:2.11.0")
implementation("androidx.activity:activity-compose:1.9.0")
implementation("androidx.compose.material3:material3:1.2.1")
implementation("androidx.compose.ui:ui:1.6.7")
implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")
```

## 后端地址

- 模拟器连接本机服务请使用：`http://10.0.2.2:8080`
- 真实手机请改成你的局域网 IP。
