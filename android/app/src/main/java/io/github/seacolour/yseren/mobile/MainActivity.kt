package io.github.seacolour.yseren.mobile

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.documentfile.provider.DocumentFile
import io.github.seacolour.yseren.mobile.share.ShareServerController
import io.github.seacolour.yseren.mobile.share.ShareServerService

class MainActivity : AppCompatActivity() {
    private lateinit var prefs: AppPrefs

    private lateinit var folderText: TextView
    private lateinit var statusText: TextView
    private lateinit var urlsText: TextView
    private lateinit var errorText: TextView
    private lateinit var startButton: Button
    private lateinit var stopButton: Button
    private lateinit var openButton: Button

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
            renderState()
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        prefs = AppPrefs(this)

        folderText = findViewById(R.id.folderText)
        statusText = findViewById(R.id.statusText)
        urlsText = findViewById(R.id.urlsText)
        errorText = findViewById(R.id.errorText)
        startButton = findViewById(R.id.startShareButton)
        stopButton = findViewById(R.id.stopShareButton)
        openButton = findViewById(R.id.openInBrowserButton)

        findViewById<Button>(R.id.selectFolderButton).setOnClickListener {
            picker.launch(prefs.loadConfig()?.treeUri)
        }

        startButton.setOnClickListener {
            if (prefs.loadConfig() == null) {
                Toast.makeText(this, "请先选择要共享的目录", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            ContextCompat.startForegroundService(
                this,
                Intent(this, ShareServerService::class.java).setAction(ShareServerService.ACTION_START),
            )
            renderState()
        }

        stopButton.setOnClickListener {
            startService(
                Intent(this, ShareServerService::class.java).setAction(ShareServerService.ACTION_STOP),
            )
            renderState()
        }

        openButton.setOnClickListener {
            if (!ShareServerController.isRunning()) {
                Toast.makeText(this, "请先启动共享服务", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            val url = ShareServerController.currentUrls(prefs.loadConfig()).firstOrNull()
            if (url == null) {
                Toast.makeText(this, "当前没有可用的局域网地址", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
        }

        renderState()
    }

    override fun onResume() {
        super.onResume()
        renderState()
    }

    private fun renderState() {
        val config = prefs.loadConfig()
        val running = ShareServerController.isRunning()
        val urls = if (running) ShareServerController.currentUrls(config) else emptyList()

        folderText.text = when {
            config == null -> "未选择目录"
            else -> "${config.displayName}\n${config.treeUri}"
        }

        statusText.text = when {
            config == null -> "请先授权一个目录"
            running -> "正在共享，端口 ${config.port}"
            else -> "已配置目录，但服务未启动"
        }

        urlsText.text = buildString {
            if (urls.isEmpty()) {
                append("启动服务后，这里会显示局域网访问地址。")
            } else {
                append("局域网访问地址：\n")
                urls.forEach { url ->
                    append(url)
                    append('\n')
                }
                append("\n主要端点：\n")
                append("/api/status\n")
                append("/api/tree?path=\n")
                append("/playlist.m3u\n")
                append("/stream/<relative-path>\n")
            }
        }.trim()

        errorText.text = ShareServerController.lastError().orEmpty()

        startButton.isEnabled = config != null && !running
        stopButton.isEnabled = running
        openButton.isEnabled = urls.isNotEmpty()
    }
}
