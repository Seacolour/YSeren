package io.github.seacolour.yseren.mobile.share

import android.content.Context
import fi.iki.elonen.NanoHTTPD
import io.github.seacolour.yseren.mobile.LanAddressUtil
import io.github.seacolour.yseren.mobile.ShareConfig

object ShareServerController {
    @Volatile
    private var server: MediaHttpServer? = null

    @Volatile
    private var activeConfig: ShareConfig? = null

    @Volatile
    private var lastError: String? = null

    @Synchronized
    fun start(context: Context, config: ShareConfig): Result<Unit> {
        stopLocked()
        return try {
            server = MediaHttpServer(context.applicationContext, config).also {
                it.start(NanoHTTPD.SOCKET_READ_TIMEOUT, false)
            }
            activeConfig = config
            lastError = null
            Result.success(Unit)
        } catch (t: Throwable) {
            stopLocked()
            lastError = t.message ?: t::class.java.simpleName
            Result.failure(t)
        }
    }

    @Synchronized
    fun stop() {
        stopLocked()
    }

    fun isRunning(): Boolean = server != null

    fun lastError(): String? = lastError

    fun currentUrls(config: ShareConfig? = activeConfig): List<String> {
        val active = config ?: return emptyList()
        return LanAddressUtil.listLanIpv4().map { ip -> "http://$ip:${active.port}/" }
    }

    private fun stopLocked() {
        server?.stop()
        server = null
        activeConfig = null
    }
}
