package io.github.seacolour.yseren.mobile

import android.Manifest
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Error
import androidx.compose.material.icons.filled.FolderOpen
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Link
import androidx.compose.material.icons.filled.OpenInBrowser
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.PowerSettingsNew
import androidx.compose.material.icons.filled.Save
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material.icons.filled.Storage
import androidx.compose.material.icons.filled.WifiTethering
import androidx.compose.material3.AssistChip
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CenterAlignedTopAppBar
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.documentfile.provider.DocumentFile
import io.github.seacolour.yseren.mobile.share.ShareServerController
import io.github.seacolour.yseren.mobile.share.ShareServerService

class MainActivity : ComponentActivity() {
    private lateinit var prefs: AppPrefs
    private var pendingStartAfterPermission = false
    private var uiState by mutableStateOf(ShareUiState())

    private val notificationPermission =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
            if (granted && pendingStartAfterPermission) {
                startShareService()
            } else if (!granted) {
                Toast.makeText(this, "需要通知权限来显示前台共享状态", Toast.LENGTH_SHORT).show()
            }
            pendingStartAfterPermission = false
            refreshState()
        }

    private val picker =
        registerForActivityResult(ActivityResultContracts.OpenDocumentTree()) { uri ->
            if (uri == null) {
                return@registerForActivityResult
            }

            try {
                contentResolver.takePersistableUriPermission(
                    uri,
                    Intent.FLAG_GRANT_READ_URI_PERMISSION,
                )
            } catch (_: SecurityException) {
                // Some providers may not support persistable permissions; best effort is enough for MVP.
            }

            val root = DocumentFile.fromTreeUri(this, uri)
            prefs.saveTree(uri, root?.name ?: "Shared Media")
            refreshState()
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        prefs = AppPrefs(this)
        refreshState()

        setContent {
            YSerenTheme {
                YSerenApp(
                    state = uiState,
                    onSelectFolder = { picker.launch(prefs.loadConfig()?.treeUri) },
                    onStart = { requestStart() },
                    onStop = { stopShareService() },
                    onOpen = { openPrimaryUrl() },
                    onCopy = { copyPrimaryUrl() },
                    onSavePort = { savePort(it) },
                )
            }
        }
    }

    override fun onResume() {
        super.onResume()
        refreshState()
    }

    private fun requestStart() {
        if (prefs.loadConfig() == null) {
            Toast.makeText(this, "请先选择要共享的目录", Toast.LENGTH_SHORT).show()
            return
        }
        if (
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            pendingStartAfterPermission = true
            notificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
            return
        }
        startShareService()
    }

    private fun startShareService() {
        ContextCompat.startForegroundService(
            this,
            Intent(this, ShareServerService::class.java).setAction(ShareServerService.ACTION_START),
        )
        refreshState()
        scheduleRefresh()
    }

    private fun stopShareService() {
        startService(
            Intent(this, ShareServerService::class.java).setAction(ShareServerService.ACTION_STOP),
        )
        refreshState()
        scheduleRefresh()
    }

    private fun openPrimaryUrl() {
        val url = uiState.urls.firstOrNull()
        if (url == null) {
            Toast.makeText(this, "当前没有可用的局域网地址", Toast.LENGTH_SHORT).show()
            return
        }
        startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
    }

    private fun copyPrimaryUrl() {
        val url = uiState.urls.firstOrNull()
        if (url == null) {
            Toast.makeText(this, "当前没有可复制的局域网地址", Toast.LENGTH_SHORT).show()
            return
        }
        val clipboard = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        clipboard.setPrimaryClip(ClipData.newPlainText("YSeren URL", url))
        Toast.makeText(this, "已复制局域网地址", Toast.LENGTH_SHORT).show()
    }

    private fun savePort(port: Int) {
        if (ShareServerController.isRunning()) {
            Toast.makeText(this, "请先停止共享服务再修改端口", Toast.LENGTH_SHORT).show()
            return
        }
        prefs.savePort(port)
        Toast.makeText(this, "端口已保存", Toast.LENGTH_SHORT).show()
        refreshState()
    }

    private fun refreshState() {
        val config = prefs.loadConfig()
        val running = ShareServerController.isRunning()
        val hasPermission = config?.let { hasTreeAccess(it.treeUri) } ?: false
        val error = ShareServerController.lastError()
        val urls = if (running) ShareServerController.currentUrls(config) else emptyList()
        uiState = ShareUiState(
            config = config,
            running = running,
            urls = urls,
            lastError = error,
            hasTreePermission = hasPermission,
        )
    }

    private fun scheduleRefresh(delayMillis: Long = 350L) {
        window.decorView.postDelayed({ refreshState() }, delayMillis)
    }

    private fun hasTreeAccess(uri: Uri): Boolean {
        val persisted = contentResolver.persistedUriPermissions.any {
            it.uri == uri && it.isReadPermission
        }
        if (persisted) {
            return true
        }
        return DocumentFile.fromTreeUri(this, uri)?.canRead() == true
    }
}

private data class ShareUiState(
    val config: ShareConfig? = null,
    val running: Boolean = false,
    val urls: List<String> = emptyList(),
    val lastError: String? = null,
    val hasTreePermission: Boolean = false,
) {
    val status: ShareStatus
        get() = when {
            config == null -> ShareStatus.NoFolder
            !hasTreePermission -> ShareStatus.PermissionLost
            running -> ShareStatus.Running
            lastError != null -> ShareStatus.Error
            else -> ShareStatus.Ready
        }
}

private enum class ShareStatus(
    val title: String,
    val description: String,
    val icon: ImageVector,
) {
    NoFolder("未选择媒体目录", "先授权一个本机目录，YSeren 才能把媒体暴露到局域网。", Icons.Filled.FolderOpen),
    Ready("准备共享", "目录已经就绪，可以启动前台共享服务。", Icons.Filled.CheckCircle),
    Running("正在共享", "服务正在前台运行，局域网设备可以访问这些地址。", Icons.Filled.WifiTethering),
    PermissionLost("目录权限失效", "Android 已经收回目录权限，请重新选择共享目录。", Icons.Filled.Error),
    Error("共享异常", "最近一次启动或运行出现错误，检查提示后再重试。", Icons.Filled.Error),
}

private enum class MainTab(
    val label: String,
    val icon: ImageVector,
) {
    Dashboard("共享", Icons.Filled.Home),
    Source("媒体源", Icons.Filled.Storage),
    Settings("设置", Icons.Filled.Settings),
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun YSerenApp(
    state: ShareUiState,
    onSelectFolder: () -> Unit,
    onStart: () -> Unit,
    onStop: () -> Unit,
    onOpen: () -> Unit,
    onCopy: () -> Unit,
    onSavePort: (Int) -> Unit,
) {
    var selectedTab by remember { mutableIntStateOf(0) }
    val tabs = MainTab.entries
    Scaffold(
        modifier = Modifier.fillMaxSize(),
        contentWindowInsets = WindowInsets.safeDrawing,
        topBar = {
            CenterAlignedTopAppBar(
                title = {
                    Text(
                        text = "YSeren",
                        fontWeight = FontWeight.SemiBold,
                    )
                },
            )
        },
        bottomBar = {
            NavigationBar {
                tabs.forEachIndexed { index, tab ->
                    NavigationBarItem(
                        selected = selectedTab == index,
                        onClick = { selectedTab = index },
                        icon = { Icon(tab.icon, contentDescription = tab.label) },
                        label = { Text(tab.label) },
                    )
                }
            }
        },
    ) { padding ->
        when (tabs[selectedTab]) {
            MainTab.Dashboard -> DashboardScreen(
                state = state,
                contentPadding = padding,
                onSelectFolder = onSelectFolder,
                onStart = onStart,
                onStop = onStop,
                onOpen = onOpen,
                onCopy = onCopy,
            )

            MainTab.Source -> SourceScreen(
                state = state,
                contentPadding = padding,
                onSelectFolder = onSelectFolder,
                onOpen = onOpen,
                onCopy = onCopy,
            )

            MainTab.Settings -> SettingsScreen(
                state = state,
                contentPadding = padding,
                onSavePort = onSavePort,
            )
        }
    }
}

@Composable
private fun DashboardScreen(
    state: ShareUiState,
    contentPadding: PaddingValues,
    onSelectFolder: () -> Unit,
    onStart: () -> Unit,
    onStop: () -> Unit,
    onOpen: () -> Unit,
    onCopy: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(contentPadding),
        contentPadding = PaddingValues(20.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        item {
            StatusCard(
                state = state,
                onStart = onStart,
                onStop = onStop,
            )
        }
        item {
            FolderCard(
                state = state,
                onSelectFolder = onSelectFolder,
            )
        }
        item {
            AddressCard(
                urls = state.urls,
                onOpen = onOpen,
                onCopy = onCopy,
            )
        }
        if (!state.lastError.isNullOrBlank()) {
            item {
                InfoCard(
                    title = "最近错误",
                    body = state.lastError,
                    icon = Icons.Filled.Error,
                    isError = true,
                )
            }
        }
    }
}

@Composable
private fun SourceScreen(
    state: ShareUiState,
    contentPadding: PaddingValues,
    onSelectFolder: () -> Unit,
    onOpen: () -> Unit,
    onCopy: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(contentPadding),
        contentPadding = PaddingValues(20.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        item {
            FolderCard(
                state = state,
                onSelectFolder = onSelectFolder,
            )
        }
        item {
            AddressCard(
                urls = state.urls,
                onOpen = onOpen,
                onCopy = onCopy,
            )
        }
        item {
            EndpointCard(port = state.config?.port ?: AppPrefs.DEFAULT_PORT)
        }
    }
}

@Composable
private fun SettingsScreen(
    state: ShareUiState,
    contentPadding: PaddingValues,
    onSavePort: (Int) -> Unit,
) {
    var portText by remember { mutableStateOf((state.config?.port ?: AppPrefs.DEFAULT_PORT).toString()) }
    LaunchedEffect(state.config?.port) {
        portText = (state.config?.port ?: AppPrefs.DEFAULT_PORT).toString()
    }
    val port = portText.toIntOrNull()
    val canSave = !state.running && port != null && port in 1024..65535

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(contentPadding),
        contentPadding = PaddingValues(20.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        item {
            SectionCard(
                title = "共享端口",
                icon = Icons.Filled.Settings,
            ) {
                Text(
                    text = "端口会影响局域网 URL。服务运行中需要先停止再修改。",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    style = MaterialTheme.typography.bodyMedium,
                )
                Spacer(modifier = Modifier.height(12.dp))
                OutlinedTextField(
                    value = portText,
                    onValueChange = { value ->
                        portText = value.filter { it.isDigit() }.take(5)
                    },
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !state.running,
                    label = { Text("HTTP 端口") },
                    supportingText = { Text("建议使用 1024 到 65535 之间的端口") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                Spacer(modifier = Modifier.height(12.dp))
                FilledTonalButton(
                    onClick = { if (port != null) onSavePort(port) },
                    enabled = canSave,
                ) {
                    Icon(Icons.Filled.Save, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("保存端口")
                }
            }
        }
        item {
            InfoCard(
                title = "前台服务",
                body = "Android 会限制后台长期运行任务。YSeren 采用常驻通知表达共享状态，停止共享也可以直接从通知里完成。",
                icon = Icons.Filled.Info,
            )
        }
    }
}

@Composable
private fun StatusCard(
    state: ShareUiState,
    onStart: () -> Unit,
    onStop: () -> Unit,
) {
    val status = state.status
    val colors = when (status) {
        ShareStatus.Running -> CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.primaryContainer,
            contentColor = MaterialTheme.colorScheme.onPrimaryContainer,
        )

        ShareStatus.Error,
        ShareStatus.PermissionLost,
        -> CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.errorContainer,
            contentColor = MaterialTheme.colorScheme.onErrorContainer,
        )

        else -> CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.secondaryContainer,
            contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
        )
    }
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(28.dp),
        colors = colors,
    ) {
        Column(
            modifier = Modifier.padding(22.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                Surface(
                    modifier = Modifier.size(52.dp),
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.surface.copy(alpha = 0.36f),
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Icon(status.icon, contentDescription = null)
                    }
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = status.title,
                        style = MaterialTheme.typography.headlineSmall,
                        fontWeight = FontWeight.SemiBold,
                    )
                    Text(
                        text = status.description,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                if (state.running) {
                    FilledTonalButton(
                        onClick = onStop,
                        colors = ButtonDefaults.filledTonalButtonColors(
                            containerColor = MaterialTheme.colorScheme.surface,
                        ),
                    ) {
                        Icon(Icons.Filled.Stop, contentDescription = null)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("停止共享")
                    }
                } else {
                    FilledTonalButton(
                        onClick = onStart,
                        enabled = state.config != null && state.hasTreePermission,
                        colors = ButtonDefaults.filledTonalButtonColors(
                            containerColor = MaterialTheme.colorScheme.surface,
                        ),
                    ) {
                        Icon(Icons.Filled.PlayArrow, contentDescription = null)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("开始共享")
                    }
                }
            }
        }
    }
}

@Composable
private fun FolderCard(
    state: ShareUiState,
    onSelectFolder: () -> Unit,
) {
    SectionCard(
        title = "媒体目录",
        icon = Icons.Filled.FolderOpen,
        trailing = {
            OutlinedButton(onClick = onSelectFolder) {
                Text(if (state.config == null) "选择" else "更换")
            }
        },
    ) {
        val config = state.config
        Text(
            text = config?.displayName ?: "未选择目录",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Medium,
        )
        Spacer(modifier = Modifier.height(6.dp))
        Text(
            text = config?.treeUri?.toString() ?: "通过 Android 系统目录选择器授权一个媒体文件夹。",
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 3,
            overflow = TextOverflow.Ellipsis,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

@Composable
private fun AddressCard(
    urls: List<String>,
    onOpen: () -> Unit,
    onCopy: () -> Unit,
) {
    SectionCard(
        title = "局域网地址",
        icon = Icons.Filled.Link,
        trailing = {
            Row {
                IconButton(onClick = onCopy, enabled = urls.isNotEmpty()) {
                    Icon(Icons.Filled.ContentCopy, contentDescription = "复制")
                }
                IconButton(onClick = onOpen, enabled = urls.isNotEmpty()) {
                    Icon(Icons.Filled.OpenInBrowser, contentDescription = "打开")
                }
            }
        },
    ) {
        if (urls.isEmpty()) {
            Text(
                text = "启动共享后，这里会显示局域网设备可访问的地址。",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                style = MaterialTheme.typography.bodyMedium,
            )
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                urls.forEach { url ->
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(14.dp),
                        color = MaterialTheme.colorScheme.surfaceContainerHighest,
                    ) {
                        Text(
                            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
                            text = url,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun EndpointCard(port: Int) {
    SectionCard(
        title = "服务端点",
        icon = Icons.Filled.PowerSettingsNew,
    ) {
        FlowRow(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            EndpointChip("/")
            EndpointChip("/api/status")
            EndpointChip("/api/tree?path=")
            EndpointChip("/playlist.m3u")
            EndpointChip("/stream/<path>")
        }
        Spacer(modifier = Modifier.height(12.dp))
        Text(
            text = "当前端口：$port",
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

@Composable
private fun EndpointChip(text: String) {
    AssistChip(
        onClick = {},
        label = { Text(text) },
    )
}

@Composable
private fun InfoCard(
    title: String,
    body: String,
    icon: ImageVector,
    isError: Boolean = false,
) {
    SectionCard(
        title = title,
        icon = icon,
        isError = isError,
    ) {
        Text(
            text = body,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

@Composable
private fun SectionCard(
    title: String,
    icon: ImageVector,
    modifier: Modifier = Modifier,
    trailing: (@Composable () -> Unit)? = null,
    isError: Boolean = false,
    content: @Composable ColumnScope.() -> Unit,
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(22.dp),
        colors = CardDefaults.cardColors(
            containerColor = if (isError) {
                MaterialTheme.colorScheme.errorContainer
            } else {
                MaterialTheme.colorScheme.surfaceContainer
            },
        ),
    ) {
        Column(
            modifier = Modifier.padding(18.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(icon, contentDescription = null)
                Spacer(modifier = Modifier.width(10.dp))
                Text(
                    modifier = Modifier.weight(1f),
                    text = title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                )
                if (trailing != null) {
                    trailing()
                }
            }
            Spacer(modifier = Modifier.height(12.dp))
            content()
        }
    }
}

@Composable
private fun YSerenTheme(content: @Composable () -> Unit) {
    val context = LocalContext.current
    val dark = isSystemInDarkTheme()
    val colorScheme = when {
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.S && dark -> dynamicDarkColorScheme(context)
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> dynamicLightColorScheme(context)
        dark -> darkColorScheme()
        else -> lightColorScheme()
    }
    MaterialTheme(
        colorScheme = colorScheme,
        typography = MaterialTheme.typography,
        content = content,
    )
}
