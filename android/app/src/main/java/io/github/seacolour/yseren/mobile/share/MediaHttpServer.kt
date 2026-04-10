package io.github.seacolour.yseren.mobile.share

import android.content.Context
import android.net.Uri
import fi.iki.elonen.NanoHTTPD
import io.github.seacolour.yseren.mobile.ShareConfig
import org.json.JSONArray
import org.json.JSONObject
import java.io.InputStream
import java.net.URLDecoder
import java.nio.charset.StandardCharsets

class MediaHttpServer(
    context: Context,
    private val config: ShareConfig,
) : NanoHTTPD(config.port) {
    private val appContext = context.applicationContext
    private val repository = DocumentTreeRepository(appContext)

    override fun serve(session: IHTTPSession): Response {
        return try {
            when {
                session.uri == "/" -> serveHomePage(session)
                session.uri == "/api/status" -> jsonResponse(buildStatusJson())
                session.uri == "/api/tree" -> serveTree(session)
                session.uri == "/playlist.m3u" -> servePlaylist(session)
                session.uri.startsWith("/stream/") -> serveStream(session)
                else -> newFixedLengthResponse(Response.Status.NOT_FOUND, MIME_PLAINTEXT, "Not found")
            }
        } catch (t: Throwable) {
            newFixedLengthResponse(
                Response.Status.INTERNAL_ERROR,
                MIME_PLAINTEXT,
                "Server error: ${t.message ?: t::class.java.simpleName}",
            )
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
        return newFixedLengthResponse(Response.Status.OK, "text/html; charset=utf-8", html)
    }

    private fun serveTree(session: IHTTPSession): Response {
        val relPath = session.parameters["path"]?.firstOrNull().orEmpty()
        val items = repository.listChildren(config.treeUri, relPath)
        val json = JSONObject()
            .put("generatedAt", System.currentTimeMillis() / 1000)
            .put("rootName", config.displayName)
            .put("path", relPath)
            .put("items", JSONArray().apply {
                items.forEach { entry ->
                    put(
                        JSONObject()
                            .put("name", entry.name)
                            .put("relPath", entry.relPath)
                            .put("type", if (entry.isDirectory) "dir" else "file")
                            .put("size", entry.size)
                            .put("lastModified", entry.lastModified)
                            .put("mimeType", entry.mimeType ?: JSONObject.NULL)
                            .put("url", if (entry.isDirectory) JSONObject.NULL else streamUrl(entry.relPath)),
                    )
                }
            })
        return jsonResponse(json)
    }

    private fun servePlaylist(session: IHTTPSession): Response {
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
                append(streamUrl(entry.relPath))
                append('\n')
            }
        }
        return newFixedLengthResponse(Response.Status.OK, "audio/x-mpegurl; charset=utf-8", body)
    }

    private fun serveStream(session: IHTTPSession): Response {
        val relPath = decodePath(session.uri.removePrefix("/stream/"))
        val file = repository.resolveMediaFile(config.treeUri, relPath)
            ?: return newFixedLengthResponse(Response.Status.NOT_FOUND, MIME_PLAINTEXT, "File not found")

        val input = appContext.contentResolver.openInputStream(file.uri)
            ?: return newFixedLengthResponse(Response.Status.INTERNAL_ERROR, MIME_PLAINTEXT, "Cannot open file")

        val totalLength = file.length().coerceAtLeast(0L)
        val mimeType = repository.guessMimeType(file)
        val range = parseRange(session.headers["range"], totalLength)

        return try {
            if (range == null) {
                val response = newFixedLengthResponse(Response.Status.OK, mimeType, input, totalLength)
                response.addHeader("Accept-Ranges", "bytes")
                response
            } else {
                skipFully(input, range.start)
                val response = newFixedLengthResponse(
                    Response.Status.PARTIAL_CONTENT,
                    mimeType,
                    LimitedInputStream(input, range.length),
                    range.length,
                )
                response.addHeader("Accept-Ranges", "bytes")
                response.addHeader("Content-Range", "bytes ${range.start}-${range.end}/$totalLength")
                response
            }
        } catch (t: Throwable) {
            input.close()
            throw t
        }
    }

    private fun buildStatusJson(): JSONObject {
        return JSONObject()
            .put("name", "YSeren Android Share")
            .put("rootName", config.displayName)
            .put("port", config.port)
            .put("urls", JSONArray(ShareServerController.currentUrls(config)))
            .put("treeUri", config.treeUri.toString())
    }

    private fun jsonResponse(body: JSONObject): Response {
        return newFixedLengthResponse(Response.Status.OK, "application/json; charset=utf-8", body.toString())
    }

    private fun streamUrl(relPath: String): String {
        val encoded = relPath.split('/')
            .filter { it.isNotEmpty() }
            .joinToString("/") { Uri.encode(it) }
        return "/stream/$encoded"
    }

    private fun decodePath(value: String): String {
        return value.split('/')
            .filter { it.isNotEmpty() }
            .joinToString("/") { URLDecoder.decode(it, StandardCharsets.UTF_8.name()) }
    }

    private fun escapeHtml(value: String): String {
        return value
            .replace("&", "&amp;")
            .replace("<", "&lt;")
            .replace(">", "&gt;")
            .replace("\"", "&quot;")
    }

    private fun parseRange(header: String?, totalLength: Long): ByteRange? {
        if (header == null || totalLength <= 0L || !header.startsWith("bytes=")) {
            return null
        }

        val candidate = header.removePrefix("bytes=").substringBefore(',').trim()
        val parts = candidate.split('-', limit = 2)
        if (parts.size != 2) {
            return null
        }

        val startPart = parts[0].trim()
        val endPart = parts[1].trim()

        return if (startPart.isEmpty()) {
            val suffixLength = endPart.toLongOrNull() ?: return null
            val start = (totalLength - suffixLength).coerceAtLeast(0L)
            ByteRange(start, totalLength - 1)
        } else {
            val start = startPart.toLongOrNull() ?: return null
            val end = if (endPart.isEmpty()) {
                totalLength - 1
            } else {
                minOf(endPart.toLongOrNull() ?: return null, totalLength - 1)
            }
            if (start < 0 || start > end || start >= totalLength) {
                return null
            }
            ByteRange(start, end)
        }
    }

    private fun skipFully(input: InputStream, bytes: Long) {
        var remaining = bytes
        while (remaining > 0) {
            val skipped = input.skip(remaining)
            if (skipped > 0) {
                remaining -= skipped
                continue
            }
            if (input.read() == -1) {
                break
            }
            remaining--
        }
    }

    private data class ByteRange(val start: Long, val end: Long) {
        val length: Long
            get() = (end - start) + 1
    }
}

private class LimitedInputStream(
    private val upstream: InputStream,
    private var remaining: Long,
) : InputStream() {
    override fun read(): Int {
        if (remaining <= 0) {
            return -1
        }
        val value = upstream.read()
        if (value != -1) {
            remaining--
        }
        return value
    }

    override fun read(buffer: ByteArray, offset: Int, length: Int): Int {
        if (remaining <= 0) {
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
