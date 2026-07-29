package io.github.seacolour.yseren.mobile.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
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
import androidx.compose.material.icons.filled.ErrorOutline
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.OpenInBrowser
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Smartphone
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material.icons.filled.Storage
import androidx.compose.material.icons.filled.VideoLibrary
import androidx.compose.material.icons.filled.Wifi
import androidx.compose.material.icons.outlined.AudioFile
import androidx.compose.material.icons.outlined.DeleteOutline
import androidx.compose.material.icons.outlined.FolderOpen
import androidx.compose.material.icons.outlined.Movie
import androidx.compose.material.icons.outlined.Source
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import io.github.seacolour.yseren.mobile.AppPrefs
import io.github.seacolour.yseren.mobile.R
import io.github.seacolour.yseren.mobile.ShareStatus
import io.github.seacolour.yseren.mobile.ShareUiState

private enum class MainTab(
    val label: String,
    val icon: ImageVector,
) {
    Dashboard("共享", Icons.Filled.Home),
    Sources("媒体源", Icons.Outlined.Source),
    Settings("设置", Icons.Filled.Settings),
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun YSerenApp(
    state: ShareUiState,
    onSelectFolder: () -> Unit,
    onSelectAndStartFolder: () -> Unit,
    onRemoveFolder: () -> Unit,
    onRefreshMedia: () -> Unit,
    onStart: () -> Unit,
    onStop: () -> Unit,
    onOpenLocal: () -> Unit,
    onCopyUrl: (String) -> Unit,
    onSavePort: (Int) -> Unit,
) {
    var selectedTab by remember { mutableIntStateOf(0) }
    var showRemoveDialog by remember { mutableStateOf(false) }
    val tabs = MainTab.entries

    Scaffold(
        modifier = Modifier.fillMaxSize(),
        containerColor = MaterialTheme.colorScheme.background,
        contentWindowInsets = WindowInsets.safeDrawing,
        topBar = { BrandTopBar(state) },
        bottomBar = {
            NavigationBar(
                containerColor = MaterialTheme.colorScheme.surface,
                tonalElevation = 3.dp,
            ) {
                tabs.forEachIndexed { index, tab ->
                    NavigationBarItem(
                        selected = selectedTab == index,
                        onClick = { selectedTab = index },
                        icon = { Icon(tab.icon, contentDescription = null) },
                        label = { Text(tab.label) },
                        colors = NavigationBarItemDefaults.colors(
                            indicatorColor = MaterialTheme.colorScheme.primaryContainer,
                        ),
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
                onSelectAndStartFolder = onSelectAndStartFolder,
                onStart = onStart,
                onStop = onStop,
                onOpenLocal = onOpenLocal,
                onCopyUrl = onCopyUrl,
                onManageSources = { selectedTab = MainTab.Sources.ordinal },
            )

            MainTab.Sources -> SourcesScreen(
                state = state,
                contentPadding = padding,
                onSelectFolder = onSelectFolder,
                onRemoveFolder = { showRemoveDialog = true },
                onRefreshMedia = onRefreshMedia,
            )

            MainTab.Settings -> SettingsScreen(
                state = state,
                contentPadding = padding,
                onSavePort = onSavePort,
            )
        }
    }

    if (showRemoveDialog) {
        AlertDialog(
            onDismissRequest = { showRemoveDialog = false },
            icon = { Icon(Icons.Outlined.DeleteOutline, contentDescription = null) },
            title = { Text("移除媒体目录？") },
            text = {
                Text("共享会停止，YSeren 将释放此目录的读取权限；手机里的原文件不会被删除。")
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        showRemoveDialog = false
                        onRemoveFolder()
                    },
                    colors = ButtonDefaults.textButtonColors(contentColor = MaterialTheme.colorScheme.error),
                ) {
                    Text("移除")
                }
            },
            dismissButton = {
                TextButton(onClick = { showRemoveDialog = false }) {
                    Text("取消")
                }
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun BrandTopBar(state: ShareUiState) {
    TopAppBar(
        title = {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Image(
                    painter = painterResource(R.drawable.yseren_logo),
                    contentDescription = "YSeren",
                    modifier = Modifier.size(36.dp),
                )
                Column {
                    Text(
                        text = "YSeren",
                        style = MaterialTheme.typography.titleMedium,
                        lineHeight = 18.sp,
                    )
                    Text(
                        text = "局域网媒体",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        },
        actions = {
            ServiceStateBadge(state.status)
            Spacer(modifier = Modifier.width(12.dp))
        },
        colors = TopAppBarDefaults.topAppBarColors(
            containerColor = MaterialTheme.colorScheme.background,
            scrolledContainerColor = MaterialTheme.colorScheme.background,
        ),
    )
}

@Composable
private fun ServiceStateBadge(status: ShareStatus) {
    val running = status == ShareStatus.Running
    val needsAttention = status == ShareStatus.Error || status == ShareStatus.PermissionLost
    val color = when {
        running -> SuccessGreen
        needsAttention -> MaterialTheme.colorScheme.error
        else -> MaterialTheme.colorScheme.onSurfaceVariant
    }
    val label = when {
        running -> "共享中"
        needsAttention -> "需处理"
        else -> "未启动"
    }
    Surface(
        shape = RoundedCornerShape(999.dp),
        color = color.copy(alpha = 0.10f),
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Box(
                modifier = Modifier
                    .size(7.dp),
                contentAlignment = Alignment.Center,
            ) {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    shape = CircleShape,
                    color = color,
                ) {}
            }
            Text(
                text = label,
                color = color,
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
            )
        }
    }
}

@Composable
private fun DashboardScreen(
    state: ShareUiState,
    contentPadding: PaddingValues,
    onSelectFolder: () -> Unit,
    onSelectAndStartFolder: () -> Unit,
    onStart: () -> Unit,
    onStop: () -> Unit,
    onOpenLocal: () -> Unit,
    onCopyUrl: (String) -> Unit,
    onManageSources: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(contentPadding),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            PageIntro(
                eyebrow = "共享",
                title = "让其他设备访问这里的媒体",
                subtitle = "文件留在手机，不上传，也不额外复制。",
            )
        }
        item {
            ShareServiceCard(
                state = state,
                onSelectFolder = onSelectFolder,
                onSelectAndStartFolder = onSelectAndStartFolder,
                onStart = onStart,
                onStop = onStop,
                onOpenLocal = onOpenLocal,
                onCopyUrl = onCopyUrl,
            )
        }
        item {
            SourceSummaryCard(
                state = state,
                onManageSources = onManageSources,
            )
        }
        if (!state.lastError.isNullOrBlank()) {
            item {
                InfoStrip(
                    icon = Icons.Filled.ErrorOutline,
                    title = "最近一次启动失败",
                    body = state.lastError,
                    error = true,
                )
            }
        }
        item {
            InfoStrip(
                icon = Icons.Filled.Security,
                title = "只在信任的局域网中使用",
                body = "访问地址没有账号验证，请不要连接不可信的公共 Wi-Fi 后开启共享。",
            )
        }
    }
}

@Composable
private fun ShareServiceCard(
    state: ShareUiState,
    onSelectFolder: () -> Unit,
    onSelectAndStartFolder: () -> Unit,
    onStart: () -> Unit,
    onStop: () -> Unit,
    onOpenLocal: () -> Unit,
    onCopyUrl: (String) -> Unit,
) {
    val presentation = statusPresentation(state.status)
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(24.dp),
        colors = CardDefaults.cardColors(containerColor = presentation.containerColor),
        border = BorderStroke(1.dp, presentation.accentColor.copy(alpha = 0.20f)),
    ) {
        Column(
            modifier = Modifier.padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                StatusIcon(presentation.icon, presentation.accentColor)
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "当前状态",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Text(
                        text = presentation.title,
                        style = MaterialTheme.typography.titleLarge,
                    )
                }
                when {
                    state.running -> {
                        OutlinedButton(
                            onClick = onStop,
                            contentPadding = PaddingValues(horizontal = 14.dp, vertical = 8.dp),
                        ) {
                            Icon(Icons.Filled.Stop, contentDescription = null, modifier = Modifier.size(18.dp))
                            Spacer(modifier = Modifier.width(6.dp))
                            Text("停止")
                        }
                    }

                    state.config != null && state.hasTreePermission -> {
                        Button(
                            onClick = onStart,
                            contentPadding = PaddingValues(horizontal = 14.dp, vertical = 8.dp),
                        ) {
                            Icon(Icons.Filled.PlayArrow, contentDescription = null, modifier = Modifier.size(18.dp))
                            Spacer(modifier = Modifier.width(6.dp))
                            Text("开始")
                        }
                    }
                }
            }

            Text(
                text = presentation.description,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                style = MaterialTheme.typography.bodyMedium,
            )

            when (state.status) {
                ShareStatus.NoFolder -> {
                    Button(
                        onClick = onSelectAndStartFolder,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Icon(Icons.Outlined.FolderOpen, contentDescription = null)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("选择媒体目录并开始共享")
                    }
                }

                ShareStatus.PermissionLost -> {
                    Button(
                        onClick = onSelectFolder,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Icon(Icons.Outlined.FolderOpen, contentDescription = null)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("重新选择媒体目录")
                    }
                }

                ShareStatus.Running -> {
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                    state.localUrl?.let { localUrl ->
                        AddressRow(
                            icon = Icons.Filled.Smartphone,
                            label = "本机预览",
                            url = localUrl,
                            actionIcon = Icons.Filled.OpenInBrowser,
                            actionDescription = "在浏览器中打开本机地址",
                            onAction = onOpenLocal,
                        )
                    }
                    if (state.lanUrls.isEmpty()) {
                        Text(
                            text = "暂未检测到局域网 IPv4 地址，请确认 Wi-Fi 已连接。",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    } else {
                        state.lanUrls.forEachIndexed { index, url ->
                            AddressRow(
                                icon = Icons.Filled.Wifi,
                                label = if (state.lanUrls.size == 1) "局域网地址" else "局域网地址 ${index + 1}",
                                url = url,
                                actionIcon = Icons.Filled.ContentCopy,
                                actionDescription = "复制局域网地址",
                                onAction = { onCopyUrl(url) },
                            )
                        }
                    }
                }

                ShareStatus.Ready,
                ShareStatus.Error,
                -> Unit
            }
        }
    }
}

private data class StatusPresentation(
    val title: String,
    val description: String,
    val icon: ImageVector,
    val accentColor: Color,
    val containerColor: Color,
)

@Composable
private fun statusPresentation(status: ShareStatus): StatusPresentation {
    return when (status) {
        ShareStatus.NoFolder -> StatusPresentation(
            title = "等待选择目录",
            description = "授权一个本机目录后，YSeren 会只扫描其中的常见音视频文件。",
            icon = Icons.Outlined.FolderOpen,
            accentColor = BrandPurple,
            containerColor = MaterialTheme.colorScheme.surface,
        )

        ShareStatus.Ready -> StatusPresentation(
            title = "可以开始共享",
            description = "媒体目录已经就绪，启动后同一局域网里的浏览器即可访问。",
            icon = Icons.Filled.CheckCircle,
            accentColor = BrandPurple,
            containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.62f),
        )

        ShareStatus.Running -> StatusPresentation(
            title = "共享运行中",
            description = "保持此设备连接当前局域网，其他设备无需安装应用。",
            icon = Icons.Filled.Wifi,
            accentColor = SuccessGreen,
            containerColor = SuccessGreen.copy(alpha = 0.075f),
        )

        ShareStatus.PermissionLost -> StatusPresentation(
            title = "目录权限已失效",
            description = "Android 已收回读取权限，请重新选择要共享的媒体目录。",
            icon = Icons.Filled.ErrorOutline,
            accentColor = MaterialTheme.colorScheme.error,
            containerColor = MaterialTheme.colorScheme.error.copy(alpha = 0.07f),
        )

        ShareStatus.Error -> StatusPresentation(
            title = "共享启动失败",
            description = "请查看下方错误提示，调整设置后可以再次尝试。",
            icon = Icons.Filled.ErrorOutline,
            accentColor = MaterialTheme.colorScheme.error,
            containerColor = MaterialTheme.colorScheme.error.copy(alpha = 0.07f),
        )
    }
}

@Composable
private fun StatusIcon(icon: ImageVector, color: Color) {
    Surface(
        modifier = Modifier.size(44.dp),
        shape = CircleShape,
        color = color.copy(alpha = 0.12f),
    ) {
        Box(contentAlignment = Alignment.Center) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(24.dp),
            )
        }
    }
}

@Composable
private fun AddressRow(
    icon: ImageVector,
    label: String,
    url: String,
    actionIcon: ImageVector,
    actionDescription: String,
    onAction: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        color = MaterialTheme.colorScheme.surface,
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
    ) {
        Row(
            modifier = Modifier.padding(start = 12.dp, top = 10.dp, bottom = 10.dp, end = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = BrandPurple,
                modifier = Modifier.size(20.dp),
            )
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = label,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text(
                    text = url,
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            IconButton(onClick = onAction) {
                Icon(actionIcon, contentDescription = actionDescription)
            }
        }
    }
}

@Composable
private fun SourceSummaryCard(
    state: ShareUiState,
    onManageSources: () -> Unit,
) {
    StandardCard {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            SquareIcon(Icons.Filled.Storage)
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = if (state.config == null) "尚未添加媒体源" else "1 个媒体源",
                    style = MaterialTheme.typography.titleMedium,
                )
                Text(
                    text = state.sourceLocation.ifBlank { "选择手机里的音视频目录" },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            TextButton(onClick = onManageSources) {
                Text(if (state.config == null) "添加" else "管理")
            }
        }
        if (state.config != null) {
            Spacer(modifier = Modifier.height(12.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                PermissionBadge(state.hasTreePermission)
                Surface(
                    shape = RoundedCornerShape(999.dp),
                    color = MaterialTheme.colorScheme.surfaceVariant,
                ) {
                    Text(
                        text = when {
                            state.mediaScan.scanning -> "正在扫描…"
                            state.mediaScan.completed -> "${state.mediaScan.totalCount} 个媒体文件"
                            else -> "等待扫描"
                        },
                        modifier = Modifier.padding(horizontal = 9.dp, vertical = 5.dp),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}

@Composable
private fun SourcesScreen(
    state: ShareUiState,
    contentPadding: PaddingValues,
    onSelectFolder: () -> Unit,
    onRemoveFolder: () -> Unit,
    onRefreshMedia: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(contentPadding),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            PageIntro(
                eyebrow = "媒体源",
                title = "选择要共享的本机目录",
                subtitle = "Android 当前使用一个目录，后续可在这里扩展更多来源。",
            )
        }
        if (state.config == null) {
            item {
                EmptySourceCard(onSelectFolder)
            }
        } else {
            item {
                SourceDetailsCard(
                    state = state,
                    onSelectFolder = onSelectFolder,
                    onRemoveFolder = onRemoveFolder,
                    onRefreshMedia = onRefreshMedia,
                )
            }
        }
        item {
            InfoStrip(
                icon = Icons.Filled.Info,
                title = "只共享媒体文件",
                body = "MP4、MP3 等常见音视频会出现在浏览器中；ZIP、文档和其他文件会自动忽略。",
            )
        }
    }
}

@Composable
private fun EmptySourceCard(onSelectFolder: () -> Unit) {
    StandardCard {
        Column(
            modifier = Modifier.fillMaxWidth(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            StatusIcon(Icons.Outlined.FolderOpen, BrandPurple)
            Text("还没有媒体目录", style = MaterialTheme.typography.titleMedium)
            Text(
                text = "使用 Android 系统目录选择器授权即可，YSeren 不会复制或上传文件。",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Button(onClick = onSelectFolder, modifier = Modifier.fillMaxWidth()) {
                Icon(Icons.Outlined.FolderOpen, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("选择媒体目录")
            }
        }
    }
}

@Composable
private fun SourceDetailsCard(
    state: ShareUiState,
    onSelectFolder: () -> Unit,
    onRemoveFolder: () -> Unit,
    onRefreshMedia: () -> Unit,
) {
    StandardCard {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            SquareIcon(Icons.Filled.Folder)
            Column(modifier = Modifier.weight(1f)) {
                Text("Android 媒体目录", style = MaterialTheme.typography.titleMedium)
                Text(
                    text = state.config?.displayName.orEmpty(),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            IconButton(
                onClick = onRefreshMedia,
                enabled = state.hasTreePermission && !state.mediaScan.scanning,
            ) {
                Icon(Icons.Filled.Refresh, contentDescription = "重新扫描媒体目录")
            }
        }
        Spacer(modifier = Modifier.height(14.dp))
        PermissionBadge(state.hasTreePermission)
        Spacer(modifier = Modifier.height(12.dp))
        Text(
            text = "位置",
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            text = state.sourceLocation.ifBlank { state.config?.displayName.orEmpty() },
            style = MaterialTheme.typography.bodyLarge,
            fontWeight = FontWeight.SemiBold,
        )

        if (state.mediaScan.scanning) {
            Spacer(modifier = Modifier.height(16.dp))
            LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
        }

        Spacer(modifier = Modifier.height(16.dp))
        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
        Spacer(modifier = Modifier.height(16.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            MediaStat(
                modifier = Modifier.weight(1f),
                icon = Icons.Outlined.Movie,
                label = "视频",
                value = if (state.mediaScan.scanning) "…" else state.mediaScan.videoCount.toString(),
            )
            MediaStat(
                modifier = Modifier.weight(1f),
                icon = Icons.Outlined.AudioFile,
                label = "音频",
                value = if (state.mediaScan.scanning) "…" else state.mediaScan.audioCount.toString(),
            )
            MediaStat(
                modifier = Modifier.weight(1f),
                icon = Icons.Filled.VideoLibrary,
                label = "总计",
                value = if (state.mediaScan.scanning) "…" else state.mediaScan.totalCount.toString(),
            )
        }

        if (!state.mediaScan.error.isNullOrBlank()) {
            Spacer(modifier = Modifier.height(12.dp))
            Text(
                text = "扫描失败：${state.mediaScan.error}",
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodySmall,
            )
        }

        Spacer(modifier = Modifier.height(16.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            OutlinedButton(
                onClick = onSelectFolder,
                modifier = Modifier.weight(1f),
            ) {
                Icon(Icons.Outlined.FolderOpen, contentDescription = null, modifier = Modifier.size(18.dp))
                Spacer(modifier = Modifier.width(6.dp))
                Text(if (state.hasTreePermission) "更换目录" else "重新授权")
            }
            TextButton(
                onClick = onRemoveFolder,
                colors = ButtonDefaults.textButtonColors(contentColor = MaterialTheme.colorScheme.error),
            ) {
                Icon(Icons.Outlined.DeleteOutline, contentDescription = null, modifier = Modifier.size(18.dp))
                Spacer(modifier = Modifier.width(4.dp))
                Text("移除")
            }
        }
    }
}

@Composable
private fun MediaStat(
    modifier: Modifier,
    icon: ImageVector,
    label: String,
    value: String,
) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        color = MaterialTheme.colorScheme.surfaceVariant,
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 12.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = BrandPurple,
                modifier = Modifier.size(20.dp),
            )
            Text(value, style = MaterialTheme.typography.titleMedium)
            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
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
    val currentPort = state.config?.port ?: AppPrefs.DEFAULT_PORT
    val validPort = port != null && port in 1024..65535
    val canSave = validPort && port != currentPort

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(contentPadding),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            PageIntro(
                eyebrow = "设置",
                title = "网络与应用",
                subtitle = "常用选项保持精简，浏览器继续负责媒体播放。",
            )
        }
        item {
            StandardCard {
                CardTitle(
                    icon = Icons.Filled.Wifi,
                    title = "网络",
                    subtitle = if (state.running) {
                        "保存端口后，当前共享会自动重启。"
                    } else {
                        "端口会出现在本机和局域网访问地址中。"
                    },
                )
                Spacer(modifier = Modifier.height(14.dp))
                OutlinedTextField(
                    value = portText,
                    onValueChange = { value -> portText = value.filter(Char::isDigit).take(5) },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text("HTTP 服务端口") },
                    supportingText = {
                        Text(
                            if (portText.isNotEmpty() && !validPort) {
                                "请输入 1024 到 65535 之间的端口"
                            } else {
                                "默认端口 ${AppPrefs.DEFAULT_PORT}"
                            },
                        )
                    },
                    isError = portText.isNotEmpty() && !validPort,
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                )
                Spacer(modifier = Modifier.height(10.dp))
                Button(
                    onClick = { port?.let(onSavePort) },
                    enabled = canSave,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (state.running) "保存并重启共享" else "保存端口")
                }
            }
        }
        item {
            StandardCard {
                CardTitle(
                    icon = Icons.Filled.Smartphone,
                    title = "前台共享服务",
                    subtitle = "常驻通知用于表达共享状态，也可以从通知中直接停止服务。",
                )
                Spacer(modifier = Modifier.height(12.dp))
                InfoStrip(
                    icon = Icons.Filled.Info,
                    title = "Android 后台限制",
                    body = "系统省电策略仍可能暂停应用。长时间共享时，建议把 YSeren 加入电池优化白名单。",
                )
            }
        }
        item {
            StandardCard {
                CardTitle(
                    icon = Icons.Filled.Info,
                    title = "关于 YSeren",
                    subtitle = "局域网媒体访问工具",
                )
                Spacer(modifier = Modifier.height(14.dp))
                KeyValueRow("当前版本", state.appVersion)
                HorizontalDivider(
                    modifier = Modifier.padding(vertical = 12.dp),
                    color = MaterialTheme.colorScheme.outlineVariant,
                )
                KeyValueRow("播放方式", "系统浏览器")
                HorizontalDivider(
                    modifier = Modifier.padding(vertical = 12.dp),
                    color = MaterialTheme.colorScheme.outlineVariant,
                )
                KeyValueRow("媒体来源", "Android 本机目录")
            }
        }
    }
}

@Composable
private fun KeyValueRow(label: String, value: String) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.SemiBold,
        )
    }
}

@Composable
private fun PageIntro(
    eyebrow: String,
    title: String,
    subtitle: String,
) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(
            text = eyebrow,
            color = BrandPurple,
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.Bold,
        )
        Text(
            text = title,
            style = MaterialTheme.typography.headlineSmall,
        )
        Text(
            text = subtitle,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun StandardCard(content: @Composable ColumnScope.() -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(22.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
    ) {
        Column(
            modifier = Modifier.padding(18.dp),
            content = content,
        )
    }
}

@Composable
private fun CardTitle(
    icon: ImageVector,
    title: String,
    subtitle: String,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        SquareIcon(icon)
        Column(modifier = Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.titleMedium)
            Text(
                text = subtitle,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun SquareIcon(icon: ImageVector) {
    Surface(
        modifier = Modifier.size(40.dp),
        shape = RoundedCornerShape(13.dp),
        color = MaterialTheme.colorScheme.primaryContainer,
    ) {
        Box(contentAlignment = Alignment.Center) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = BrandPurple,
                modifier = Modifier.size(21.dp),
            )
        }
    }
}

@Composable
private fun PermissionBadge(hasPermission: Boolean) {
    val color = if (hasPermission) SuccessGreen else MaterialTheme.colorScheme.error
    Surface(
        shape = RoundedCornerShape(999.dp),
        color = color.copy(alpha = 0.10f),
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 9.dp, vertical = 5.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(5.dp),
        ) {
            Icon(
                imageVector = if (hasPermission) Icons.Filled.CheckCircle else Icons.Filled.ErrorOutline,
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(15.dp),
            )
            Text(
                text = if (hasPermission) "目录可读取" else "权限已失效",
                style = MaterialTheme.typography.labelSmall,
                color = color,
                fontWeight = FontWeight.SemiBold,
            )
        }
    }
}

@Composable
private fun InfoStrip(
    icon: ImageVector,
    title: String,
    body: String,
    error: Boolean = false,
) {
    val color = if (error) MaterialTheme.colorScheme.error else BrandPurple
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        color = color.copy(alpha = 0.065f),
        border = BorderStroke(1.dp, color.copy(alpha = 0.14f)),
    ) {
        Row(
            modifier = Modifier.padding(13.dp),
            verticalAlignment = Alignment.Top,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(20.dp),
            )
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.labelLarge,
                )
                Text(
                    text = body,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
