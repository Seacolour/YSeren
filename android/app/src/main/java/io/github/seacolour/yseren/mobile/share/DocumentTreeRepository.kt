package io.github.seacolour.yseren.mobile.share

import android.content.Context
import android.net.Uri
import android.webkit.MimeTypeMap
import androidx.documentfile.provider.DocumentFile
import java.util.ArrayDeque
import java.util.Locale

data class ShareEntry(
    val name: String,
    val relPath: String,
    val isDirectory: Boolean,
    val size: Long,
    val lastModified: Long,
    val mimeType: String?,
)

class DocumentTreeRepository(private val context: Context) {
    fun resolve(rootUri: Uri, relPath: String): DocumentFile? {
        val root = DocumentFile.fromTreeUri(context, rootUri) ?: return null
        val clean = relPath.trim('/').replace('\\', '/')
        if (clean.isEmpty()) {
            return root
        }

        var current: DocumentFile = root
        for (segment in clean.split('/')) {
            current = current.findFile(segment) ?: return null
        }
        return current
    }

    fun listChildren(rootUri: Uri, relPath: String): List<ShareEntry> {
        val directory = resolve(rootUri, relPath) ?: return emptyList()
        if (!directory.isDirectory) {
            return emptyList()
        }

        return directory.listFiles()
            .mapNotNull { child ->
                if (!child.canRead()) {
                    return@mapNotNull null
                }
                val childName = child.name ?: return@mapNotNull null
                val childRelPath = buildRelativePath(relPath, childName)
                when {
                    child.isDirectory -> child.toEntry(childRelPath)
                    isMediaFile(childName) -> child.toEntry(childRelPath)
                    else -> null
                }
            }
            .sortedWith(compareBy<ShareEntry> { !it.isDirectory }.thenBy { it.name.lowercase(Locale.ROOT) })
    }

    fun resolveMediaFile(rootUri: Uri, relPath: String): DocumentFile? {
        val file = resolve(rootUri, relPath) ?: return null
        if (!file.isFile) {
            return null
        }
        val name = file.name ?: return null
        return if (isMediaFile(name)) file else null
    }

    fun collectMediaFiles(rootUri: Uri): List<ShareEntry> {
        val root = DocumentFile.fromTreeUri(context, rootUri) ?: return emptyList()
        val results = mutableListOf<ShareEntry>()
        val queue = ArrayDeque<Pair<DocumentFile, String>>()
        queue.addLast(root to "")

        while (queue.isNotEmpty()) {
            val (doc, relPath) = queue.removeFirst()
            for (child in doc.listFiles()) {
                if (!child.canRead()) {
                    continue
                }
                val name = child.name ?: continue
                val childRelPath = buildRelativePath(relPath, name)
                when {
                    child.isDirectory -> queue.addLast(child to childRelPath)
                    isMediaFile(name) -> results += child.toEntry(childRelPath)
                }
            }
        }

        return results.sortedBy { it.relPath.lowercase(Locale.ROOT) }
    }

    fun guessMimeType(file: DocumentFile): String {
        file.type?.let { return it }
        val ext = file.name
            ?.substringAfterLast('.', missingDelimiterValue = "")
            ?.lowercase(Locale.ROOT)
            .orEmpty()
        return MimeTypeMap.getSingleton().getMimeTypeFromExtension(ext) ?: "application/octet-stream"
    }

    private fun DocumentFile.toEntry(relPath: String): ShareEntry {
        return ShareEntry(
            name = name.orEmpty(),
            relPath = relPath,
            isDirectory = isDirectory,
            size = if (isDirectory) 0L else length(),
            lastModified = lastModified(),
            mimeType = type,
        )
    }

    private fun buildRelativePath(parent: String, name: String): String {
        val cleanParent = parent.trim('/')
        return if (cleanParent.isEmpty()) name else "$cleanParent/$name"
    }

    private fun isMediaFile(name: String): Boolean {
        val extValue = name.substringAfterLast('.', missingDelimiterValue = "")
        if (extValue.isEmpty()) {
            return false
        }
        val ext = ".${extValue.lowercase(Locale.ROOT)}"
        return ext in MEDIA_EXTENSIONS
    }

    companion object {
        private val MEDIA_EXTENSIONS = setOf(
            ".mp4",
            ".mkv",
            ".webm",
            ".mov",
            ".m4v",
            ".avi",
            ".mp3",
            ".flac",
            ".wav",
            ".aac",
            ".ogg",
            ".m4a",
            ".wma",
            ".opus",
            ".oga",
            ".weba",
            ".mka",
        )
    }
}
