package io.github.seacolour.yseren.mobile.share

import android.content.Context
import android.net.Uri
import fi.iki.elonen.NanoHTTPD
import io.github.seacolour.yseren.mobile.ShareConfig
import org.json.JSONArray
import org.json.JSONObject
import java.io.ByteArrayInputStream
import java.io.IOException
import java.io.InputStream
import java.nio.charset.StandardCharsets
import java.util.Locale

class MediaHttpServer(
    context: Context,
    private val config: ShareConfig,
) : NanoHTTPD(config.port) {
    private val appContext = context.applicationContext
    private val repository = DocumentTreeRepository(appContext)

    override fun serve(session: IHTTPSession): Response {
        return try {
            when {
                session.uri == "/api/status" -> serveStatus(session)
                session.uri == "/api/tree" -> serveTree(session)
                session.uri == "/api/version" -> serveVersion(session)
                session.uri == "/playlist.m3u" -> servePlaylist(session)
                session.uri.startsWith("/stream/") -> serveStream(session)
                else -> serveFrontend(session)
            }
        } catch (t: Throwable) {
            newFixedLengthResponse(
                Response.Status.INTERNAL_ERROR,
                MIME_PLAINTEXT,
                "Server error: ${t.message ?: t::class.java.simpleName}",
            )
        }
    }

    private fun serveStatus(session: IHTTPSession): Response {
        if (session.method != Method.GET) {
            return methodNotAllowed("GET")
        }
        return jsonResponse(buildStatusJson())
    }

    private fun serveVersion(session: IHTTPSession): Response {
        if (session.method != Method.GET) {
            return methodNotAllowed("GET")
        }
        return jsonResponse(
            JSONObject()
                .put("version", appVersionName())
                .put("releaseUrl", RELEASE_PAGE_URL),
        )
    }

    private fun serveFrontend(session: IHTTPSession): Response {
        if (session.method != Method.GET && session.method != Method.HEAD) {
            return methodNotAllowed("GET, HEAD")
        }

        val assetPath = frontendAssetPath(session.uri)
        if (assetPath != null) {
            assetResponse(assetPath, session.method == Method.HEAD)?.let { return it }
        }

        assetResponse("index.html", session.method == Method.HEAD)?.let { return it }

        return if (session.uri == "/") {
            serveHomePage(session)
        } else {
            newFixedLengthResponse(Response.Status.NOT_FOUND, MIME_PLAINTEXT, "Not found")
        }
    }

    private fun frontendAssetPath(uri: String): String? {
        val clean = uri.substringBefore('?')
            .trimStart('/')
            .ifBlank { "index.html" }
        val parts = clean.split('/').filter { it.isNotBlank() }
        if (parts.any { it == "." || it == ".." }) {
            return null
        }
        return parts.joinToString("/")
    }

    private fun assetResponse(assetPath: String, headOnly: Boolean): Response? {
        return try {
            val bytes = appContext.assets.open(assetPath).use { it.readBytes() }
            fixedLengthResponse(
                status = Response.Status.OK,
                mimeType = mimeForAsset(assetPath),
                bytes = bytes,
                headOnly = headOnly,
            )
        } catch (_: IOException) {
            null
        }
    }

    private fun mimeForAsset(assetPath: String): String {
        return when (assetPath.substringAfterLast('.', "").lowercase(Locale.ROOT)) {
            "html" -> "text/html; charset=utf-8"
            "js" -> "application/javascript; charset=utf-8"
            "css" -> "text/css; charset=utf-8"
            "json" -> "application/json; charset=utf-8"
            "svg" -> "image/svg+xml"
            "png" -> "image/png"
            "jpg", "jpeg" -> "image/jpeg"
            "webp" -> "image/webp"
            "ico" -> "image/x-icon"
            "wasm" -> "application/wasm"
            "map" -> "application/json; charset=utf-8"
            else -> "application/octet-stream"
        }
    }

    private fun serveHomePage(session: IHTTPSession): Response {
        val urls = ShareServerController.currentUrls(config)
        val html = buildString {
            append("<!doctype html><html><head><meta charset=\"utf-8\">")
            append("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
            append("<title>YSeren Android Share</title></head><body>")
            append("<h1>YSeren Android Share</h1>")
            append("<p>目录：${escapeHtml(config.displayName)}</p>")
            append("<p>端口：${config.port}</p>")
            append("<h2>局域网地址</h2><ul>")
            urls.forEach { url ->
                append("<li><a href=\"$url\">$url</a></li>")
            }
            append("</ul>")
            append("<h2>端点</h2><ul>")
            append("<li><a href=\"/api/status\">/api/status</a></li>")
            append("<li><a href=\"/api/tree\">/api/tree</a></li>")
            append("<li><a href=\"/playlist.m3u\">/playlist.m3u</a></li>")
            append("</ul>")
            append("<p>当前请求 Host：${escapeHtml(session.headers["host"].orEmpty())}</p>")
            append("</body></html>")
        }
        return fixedLengthResponse(
            status = Response.Status.OK,
            mimeType = "text/html; charset=utf-8",
            bytes = html.toByteArray(StandardCharsets.UTF_8),
            headOnly = session.method == Method.HEAD,
        )
    }

    private fun serveTree(session: IHTTPSession): Response {
        if (session.method != Method.GET) {
            return methodNotAllowed("GET")
        }
        if (session.parameters.containsKey("path")) {
            return serveTreeListing(session)
        }

        val requestedSource = session.parameters["source"]?.firstOrNull().orEmpty().trim()
        if (requestedSource.isNotEmpty() && requestedSource != ANDROID_SOURCE) {
            return jsonError(Response.Status.BAD_REQUEST, "unknown source: $requestedSource")
        }

        val query = session.parameters["q"]?.firstOrNull().orEmpty()
        val sourceRoot = repository.buildMediaTree(
            rootUri = config.treeUri,
            rootName = ANDROID_SOURCE,
            source = ANDROID_SOURCE,
        )
        val filteredSource = repository.filterTree(sourceRoot, query)
        val root = if (requestedSource.isNotEmpty()) {
            filteredSource ?: sourceRoot.copy(children = emptyList())
        } else {
            ShareTreeNode(
                type = "dir",
                name = "root",
                relPath = "",
                children = listOfNotNull(filteredSource),
            )
        }
        return jsonResponse(
            TreeContractJson.response(
                root = root,
                generatedAt = System.currentTimeMillis() / 1000L,
                source = requestedSource,
            ),
        )
    }

    private fun serveTreeListing(session: IHTTPSession): Response {
        val relPath = session.parameters["path"]?.firstOrNull().orEmpty()
        val items = repository.listChildren(config.treeUri, relPath)
        val json = JSONObject()
            .put("generatedAt", System.currentTimeMillis() / 1000L)
            .put("rootName", config.displayName)
            .put("source", ANDROID_SOURCE)
            .put("path", relPath)
            .put("items", JSONArray().apply {
                items.forEach { entry ->
                    put(
                        JSONObject()
                            .put("name", entry.name)
                            .put("relPath", entry.relPath)
                            .put("source", ANDROID_SOURCE)
                            .put("type", if (entry.isDirectory) "dir" else "file")
                            .put("size", entry.size)
                            .put("modTime", entry.modTime)
                            .put("lastModified", entry.modTime * 1000L)
                            .put("mimeType", entry.mimeType ?: JSONObject.NULL)
                            .put(
                                "url",
                                if (entry.isDirectory) {
                                    JSONObject.NULL
                                } else {
                                    streamUrl(ANDROID_SOURCE, entry.relPath)
                                },
                            ),
                    )
                }
            })
        return jsonResponse(json)
    }

    private fun servePlaylist(session: IHTTPSession): Response {
        if (session.method != Method.GET && session.method != Method.HEAD) {
            return methodNotAllowed("GET, HEAD")
        }
        val host = session.headers["host"].orEmpty().ifBlank { "localhost:${config.port}" }
        val baseUrl = "http://$host"
        val mediaFiles = repository.collectMediaFiles(config.treeUri)
        val body = buildString {
            append("#EXTM3U\n")
            mediaFiles.forEach { entry ->
                append("#EXTINF:-1,")
                append(entry.name)
                append('\n')
                append(baseUrl)
                append(streamUrl(ANDROID_SOURCE, entry.relPath))
                append('\n')
            }
        }
        return fixedLengthResponse(
            status = Response.Status.OK,
            mimeType = "audio/x-mpegurl; charset=utf-8",
            bytes = body.toByteArray(StandardCharsets.UTF_8),
            headOnly = session.method == Method.HEAD,
        )
    }

    private fun serveStream(session: IHTTPSession): Response {
        if (session.method != Method.GET && session.method != Method.HEAD) {
            return methodNotAllowed("GET, HEAD")
        }

        val relPath = streamRelativePath(session.uri)
            ?: return newFixedLengthResponse(Response.Status.NOT_FOUND, MIME_PLAINTEXT, "File not found")
        val file = repository.resolveMediaFile(config.treeUri, relPath)
            ?: return newFixedLengthResponse(Response.Status.NOT_FOUND, MIME_PLAINTEXT, "File not found")

        val totalLength = file.length().coerceAtLeast(0L)
        val mimeType = repository.guessMimeType(file)
        return when (val selection = HttpRange.parse(session.headers["range"], totalLength)) {
            RangeSelection.Unsatisfiable -> rangeNotSatisfiable(totalLength)
            RangeSelection.Full -> {
                if (session.method == Method.HEAD) {
                    streamHeaders(
                        newFixedLengthResponse(
                            Response.Status.OK,
                            mimeType,
                            ByteArrayInputStream(EMPTY_BYTES),
                            totalLength,
                        ),
                    )
                } else {
                    val input = openMediaInput(file.uri) ?: return newFixedLengthResponse(
                        Response.Status.INTERNAL_ERROR,
                        MIME_PLAINTEXT,
                        "Cannot open file",
                    )
                    streamHeaders(
                        newFixedLengthResponse(Response.Status.OK, mimeType, input, totalLength),
                    )
                }
            }

            is RangeSelection.Partial -> {
                if (session.method == Method.HEAD) {
                    streamHeaders(
                        newFixedLengthResponse(
                            Response.Status.PARTIAL_CONTENT,
                            mimeType,
                            ByteArrayInputStream(EMPTY_BYTES),
                            selection.length,
                        ),
                        contentRange = "bytes ${selection.start}-${selection.end}/$totalLength",
                    )
                } else {
                    val input = openMediaInput(file.uri) ?: return newFixedLengthResponse(
                        Response.Status.INTERNAL_ERROR,
                        MIME_PLAINTEXT,
                        "Cannot open file",
                    )
                    try {
                        if (!skipFully(input, selection.start)) {
                            input.close()
                            return rangeNotSatisfiable(totalLength)
                        }
                        streamHeaders(
                            newFixedLengthResponse(
                                Response.Status.PARTIAL_CONTENT,
                                mimeType,
                                LimitedInputStream(input, selection.length),
                                selection.length,
                            ),
                            contentRange = "bytes ${selection.start}-${selection.end}/$totalLength",
                        )
                    } catch (t: Throwable) {
                        input.close()
                        throw t
                    }
                }
            }
        }
    }

    private fun openMediaInput(uri: Uri): InputStream? {
        return appContext.contentResolver.openInputStream(uri)
    }

    private fun rangeNotSatisfiable(totalLength: Long): Response {
        return streamHeaders(
            newFixedLengthResponse(
                Response.Status.RANGE_NOT_SATISFIABLE,
                MIME_PLAINTEXT,
                ByteArrayInputStream(EMPTY_BYTES),
                0L,
            ),
            contentRange = "bytes */$totalLength",
        )
    }

    private fun streamHeaders(response: Response, contentRange: String? = null): Response {
        response.addHeader("Accept-Ranges", "bytes")
        response.addHeader("X-Content-Type-Options", "nosniff")
        contentRange?.let { response.addHeader("Content-Range", it) }
        return response
    }

    private fun buildStatusJson(): JSONObject {
        return JSONObject()
            .put("state", "running")
            .put("name", "YSeren Android Share")
            .put("source", ANDROID_SOURCE)
            .put("rootName", config.displayName)
            .put("port", config.port)
            .put("urls", JSONArray(ShareServerController.currentUrls(config)))
    }

    @Suppress("DEPRECATION")
    private fun appVersionName(): String {
        return appContext.packageManager
            .getPackageInfo(appContext.packageName, 0)
            .versionName
            .orEmpty()
            .ifBlank { "dev" }
    }

    private fun jsonResponse(body: JSONObject): Response {
        return newFixedLengthResponse(Response.Status.OK, JSON_MIME, body.toString())
    }

    private fun jsonError(status: Response.Status, message: String): Response {
        return newFixedLengthResponse(status, JSON_MIME, JSONObject().put("error", message).toString())
    }

    private fun methodNotAllowed(allow: String): Response {
        return newFixedLengthResponse(
            Response.Status.METHOD_NOT_ALLOWED,
            JSON_MIME,
            JSONObject().put("error", "method not allowed").toString(),
        ).also { it.addHeader("Allow", allow) }
    }

    private fun fixedLengthResponse(
        status: Response.Status,
        mimeType: String,
        bytes: ByteArray,
        headOnly: Boolean,
    ): Response {
        return newFixedLengthResponse(
            status,
            mimeType,
            ByteArrayInputStream(if (headOnly) EMPTY_BYTES else bytes),
            bytes.size.toLong(),
        )
    }

    private fun streamUrl(source: String, relPath: String): String {
        val encoded = relPath.split('/')
            .filter { it.isNotEmpty() }
            .joinToString("/") { Uri.encode(it) }
        return "/stream/${Uri.encode(source)}/$encoded"
    }

    private fun streamRelativePath(uri: String): String? {
        val remainder = uri.removePrefix("/stream/")
        val segments = remainder.split('/')
        if (segments.isEmpty() || segments.any { it.isEmpty() || it == "." || it == ".." || '\\' in it || '\u0000' in it }) {
            return null
        }
        val relativeSegments = if (segments.size > 1 && segments.first() == ANDROID_SOURCE) {
            segments.drop(1)
        } else {
            segments
        }
        return relativeSegments.takeIf { it.isNotEmpty() }?.joinToString("/")
    }

    private fun escapeHtml(value: String): String {
        return value
            .replace("&", "&amp;")
            .replace("<", "&lt;")
            .replace(">", "&gt;")
            .replace("\"", "&quot;")
    }

    private fun skipFully(input: InputStream, bytes: Long): Boolean {
        var remaining = bytes
        while (remaining > 0L) {
            val skipped = input.skip(remaining)
            if (skipped > 0L) {
                remaining -= skipped
                continue
            }
            if (input.read() == -1) {
                return false
            }
            remaining--
        }
        return true
    }

    private companion object {
        const val JSON_MIME = "application/json; charset=utf-8"
        const val RELEASE_PAGE_URL = "https://github.com/Seacolour/YSeren/releases"
        val EMPTY_BYTES = ByteArray(0)
    }
}

private class LimitedInputStream(
    private val upstream: InputStream,
    private var remaining: Long,
) : InputStream() {
    override fun read(): Int {
        if (remaining <= 0L) {
            return -1
        }
        val value = upstream.read()
        if (value != -1) {
            remaining--
        }
        return value
    }

    override fun read(buffer: ByteArray, offset: Int, length: Int): Int {
        if (remaining <= 0L) {
            return -1
        }
        val toRead = minOf(length.toLong(), remaining).toInt()
        val count = upstream.read(buffer, offset, toRead)
        if (count > 0) {
            remaining -= count.toLong()
        }
        return count
    }

    override fun close() {
        upstream.close()
    }
}
