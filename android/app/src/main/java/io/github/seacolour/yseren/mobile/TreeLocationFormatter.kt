package io.github.seacolour.yseren.mobile

import android.net.Uri
import android.provider.DocumentsContract

internal object TreeLocationFormatter {
    fun describe(uri: Uri, fallbackName: String): String {
        val documentId = try {
            DocumentsContract.getTreeDocumentId(uri)
        } catch (_: Throwable) {
            return fallbackName
        }
        if (documentId.isBlank()) {
            return fallbackName
        }

        val separator = documentId.indexOf(':')
        if (separator < 0) {
            return fallbackName
        }
        val volume = documentId.substring(0, separator)
        val relativePath = documentId.substring(separator + 1)
        val rootName = when (volume.lowercase()) {
            "primary" -> "内部存储"
            "home" -> "文档"
            "raw" -> "本机存储"
            else -> "存储设备"
        }
        val parts = relativePath
            .replace('\\', '/')
            .split('/')
            .filter { it.isNotBlank() }
        return (listOf(rootName) + parts).joinToString(" / ")
    }
}
