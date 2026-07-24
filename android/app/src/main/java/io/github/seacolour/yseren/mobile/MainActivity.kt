package io.github.seacolour.yseren.mobile

import android.Manifest
import android.content.ActivityNotFoundException
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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.content.ContextCompat
import androidx.documentfile.provider.DocumentFile
import io.github.seacolour.yseren.mobile.share.DocumentTreeRepository
import io.github.seacolour.yseren.mobile.share.ShareServerController
import io.github.seacolour.yseren.mobile.share.ShareServerService
import io.github.seacolour.yseren.mobile.ui.YSerenApp
import io.github.seacolour.yseren.mobile.ui.YSerenTheme
import java.util.concurrent.Executors

class MainActivity : ComponentActivity() {
    private lateinit var prefs: AppPrefs
    private val scanExecutor = Executors.newSingleThreadExecutor()
    private val repository by lazy { DocumentTreeRepository(applicationContext) }

    private var pendingStartAfterNotificationPermission = false
    private var pendingStartAfterFolderSelection = false
    private var scanGeneration = 0
    private var mediaScan = MediaScanUiState()
    private var uiState by mutableStateOf(ShareUiState())

    private val notificationPermission =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
            if (granted && pendingStartAfterNotificationPermission) {
                startShareService()
            } else if (!granted) {
                Toast.makeText(this, "需要通知权限来显示前台共享状态", Toast.LENGTH_SHORT).show()
            }
            pendingStartAfterNotificationPermission = false
            refreshState()
        }

    private val folderPicker =
        registerForActivityResult(ActivityResultContracts.OpenDocumentTree()) { uri ->
            val shouldStart = pendingStartAfterFolderSelection
            pendingStartAfterFolderSelection = false
            if (uri == null) {
                return@registerForActivityResult
            }

            val previousConfig = prefs.loadConfig()
            val wasRunning = ShareServerController.isRunning()
            try {
                contentResolver.takePersistableUriPermission(
                    uri,
                    Intent.FLAG_GRANT_READ_URI_PERMISSION,
                )
            } catch (_: SecurityException) {
                // Some providers only keep the current grant. The read check below remains authoritative.
            }

            val root = DocumentFile.fromTreeUri(this, uri)
            prefs.saveTree(uri, root?.name ?: "Shared Media")
            if (previousConfig?.treeUri != null && previousConfig.treeUri != uri) {
                releaseTreePermission(previousConfig.treeUri)
            }

            scanGeneration++
            mediaScan = MediaScanUiState()
            refreshState()
            requestMediaScan(force = true)
            when {
                shouldStart -> requestStart()
                wasRunning -> startShareService()
            }
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        prefs = AppPrefs(this)
        refreshState()
        requestMediaScan(force = false)

        setContent {
            YSerenTheme {
                YSerenApp(
                    state = uiState,
                    onSelectFolder = { launchFolderPicker(startAfterSelection = false) },
                    onSelectAndStartFolder = { launchFolderPicker(startAfterSelection = true) },
                    onRemoveFolder = { removeFolder() },
                    onRefreshMedia = { requestMediaScan(force = true) },
                    onStart = { requestStart() },
                    onStop = { stopShareService() },
                    onOpenLocal = { openLocalUrl() },
                    onCopyUrl = { copyUrl(it) },
                    onSavePort = { savePort(it) },
                )
            }
        }
    }

    override fun onResume() {
        super.onResume()
        if (::prefs.isInitialized) {
            refreshState()
            requestMediaScan(force = false)
        }
    }

    override fun onDestroy() {
        scanGeneration++
        scanExecutor.shutdownNow()
        super.onDestroy()
    }

    private fun launchFolderPicker(startAfterSelection: Boolean) {
        pendingStartAfterFolderSelection = startAfterSelection
        folderPicker.launch(prefs.loadConfig()?.treeUri)
    }

    private fun requestStart() {
        val config = prefs.loadConfig()
        if (config == null) {
            Toast.makeText(this, "请先选择要共享的目录", Toast.LENGTH_SHORT).show()
            return
        }
        if (!hasTreeAccess(config.treeUri)) {
            Toast.makeText(this, "目录读取权限已失效，请重新选择目录", Toast.LENGTH_SHORT).show()
            refreshState()
            return
        }
        if (
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            pendingStartAfterNotificationPermission = true
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
        scheduleRefresh(250L)
        scheduleRefresh(800L)
    }

    private fun stopShareService() {
        startService(
            Intent(this, ShareServerService::class.java).setAction(ShareServerService.ACTION_STOP),
        )
        refreshState()
        scheduleRefresh(250L)
    }

    private fun openLocalUrl() {
        val url = uiState.localUrl
        if (url == null) {
            Toast.makeText(this, "请先启动共享", Toast.LENGTH_SHORT).show()
            return
        }
        try {
            startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
        } catch (_: ActivityNotFoundException) {
            Toast.makeText(this, "没有找到可用的浏览器", Toast.LENGTH_SHORT).show()
        }
    }

    private fun copyUrl(url: String) {
        val clipboard = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        clipboard.setPrimaryClip(ClipData.newPlainText("YSeren URL", url))
        Toast.makeText(this, "地址已复制", Toast.LENGTH_SHORT).show()
    }

    private fun savePort(port: Int) {
        if (port !in 1024..65535) {
            Toast.makeText(this, "端口需要在 1024 到 65535 之间", Toast.LENGTH_SHORT).show()
            return
        }
        val currentPort = prefs.loadConfig()?.port ?: AppPrefs.DEFAULT_PORT
        if (port == currentPort) {
            return
        }

        val wasRunning = ShareServerController.isRunning()
        prefs.savePort(port)
        if (wasRunning) {
            startShareService()
            Toast.makeText(this, "端口已保存，共享正在重启", Toast.LENGTH_SHORT).show()
        } else {
            refreshState()
            Toast.makeText(this, "端口已保存", Toast.LENGTH_SHORT).show()
        }
    }

    private fun removeFolder() {
        val config = prefs.loadConfig() ?: return
        if (ShareServerController.isRunning()) {
            stopShareService()
        }
        prefs.clearTree()
        releaseTreePermission(config.treeUri)
        scanGeneration++
        mediaScan = MediaScanUiState()
        refreshState()
        Toast.makeText(this, "媒体目录已移除", Toast.LENGTH_SHORT).show()
    }

    private fun releaseTreePermission(uri: Uri) {
        try {
            contentResolver.releasePersistableUriPermission(
                uri,
                Intent.FLAG_GRANT_READ_URI_PERMISSION,
            )
        } catch (_: SecurityException) {
            // The provider may not expose a persistable grant; there is nothing else to release.
        }
    }

    private fun requestMediaScan(force: Boolean) {
        val config = prefs.loadConfig() ?: return
        val sourceKey = config.treeUri.toString()
        if (!hasTreeAccess(config.treeUri)) {
            return
        }
        if (
            !force &&
            mediaScan.sourceKey == sourceKey &&
            (mediaScan.scanning || mediaScan.completed)
        ) {
            return
        }

        val generation = ++scanGeneration
        mediaScan = MediaScanUiState(sourceKey = sourceKey, scanning = true)
        refreshState()
        scanExecutor.execute {
            val result = runCatching { repository.scanSummary(config.treeUri) }
            runOnUiThread {
                if (generation != scanGeneration || isDestroyed) {
                    return@runOnUiThread
                }
                if (prefs.loadConfig()?.treeUri?.toString() != sourceKey) {
                    return@runOnUiThread
                }
                mediaScan = result.fold(
                    onSuccess = { summary ->
                        MediaScanUiState(
                            sourceKey = sourceKey,
                            completed = true,
                            videoCount = summary.videoCount,
                            audioCount = summary.audioCount,
                        )
                    },
                    onFailure = { error ->
                        MediaScanUiState(
                            sourceKey = sourceKey,
                            completed = true,
                            error = error.message ?: "无法读取目录",
                        )
                    },
                )
                refreshState()
            }
        }
    }

    private fun refreshState() {
        val config = prefs.loadConfig()
        val running = ShareServerController.isRunning()
        val hasPermission = config?.let { hasTreeAccess(it.treeUri) } ?: false
        val activeScan = if (mediaScan.sourceKey == config?.treeUri?.toString()) {
            mediaScan
        } else {
            MediaScanUiState()
        }
        uiState = ShareUiState(
            config = config,
            running = running,
            localUrl = if (running && config != null) "http://127.0.0.1:${config.port}/" else null,
            lanUrls = if (running) ShareServerController.currentUrls(config) else emptyList(),
            lastError = ShareServerController.lastError(),
            hasTreePermission = hasPermission,
            sourceLocation = config?.let { TreeLocationFormatter.describe(it.treeUri, it.displayName) }.orEmpty(),
            mediaScan = activeScan,
            appVersion = appVersionName(),
        )
    }

    private fun scheduleRefresh(delayMillis: Long) {
        window.decorView.postDelayed(
            {
                if (!isDestroyed) {
                    refreshState()
                }
            },
            delayMillis,
        )
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

    @Suppress("DEPRECATION")
    private fun appVersionName(): String {
        return packageManager
            .getPackageInfo(packageName, 0)
            .versionName
            .orEmpty()
            .ifBlank { "dev" }
    }
}
