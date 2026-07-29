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
    val modTime: Long,
    val mimeType: String?,
)

data class MediaScanSummary(
    val videoCount: Int,
    val audioCount: Int,
) {
    val totalCount: Int
        get() = videoCount + audioCount
}

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

    fun scanSummary(rootUri: Uri): MediaScanSummary {
        val media = collectMediaFiles(rootUri)
        val audioCount = media.count { isAudioFile(it.name) }
        return MediaScanSummary(
            videoCount = media.size - audioCount,
            audioCount = audioCount,
        )
    }

    fun buildMediaTree(rootUri: Uri, rootName: String, source: String): ShareTreeNode {
        val root = DocumentFile.fromTreeUri(context, rootUri)
            ?: return ShareTreeNode(type = "dir", name = rootName, relPath = "", source = source)
        return buildTreeNode(root, rootName, "", source)
    }

    fun filterTree(node: ShareTreeNode, query: String): ShareTreeNode? {
        val q = query.trim().lowercase(Locale.ROOT)
        if (q.isEmpty()) {
            return node
        }

        val matched = node.name.lowercase(Locale.ROOT).contains(q) ||
            node.relPath.lowercase(Locale.ROOT).contains(q)
        val keptChildren = node.children.mapNotNull { filterTree(it, q) }

        return if (matched || keptChildren.isNotEmpty()) {
            node.copy(children = keptChildren)
        } else {
            null
        }
    }

    fun guessMimeType(file: DocumentFile): String {
        file.type?.let { return it }
        val ext = file.name
            ?.substringAfterLast('.', missingDelimiterValue = "")
            ?.lowercase(Locale.ROOT)
            .orEmpty()
        return MimeTypeMap.getSingleton().getMimeTypeFromExtension(ext) ?: "application/octet-stream"
    }

    private fun buildTreeNode(
        document: DocumentFile,
        displayName: String,
        relPath: String,
        source: String,
    ): ShareTreeNode {
        if (!document.isDirectory) {
            return ShareTreeNode(
                type = "file",
                name = displayName,
                relPath = relPath,
                source = source,
                url = streamUrl(source, relPath),
                size = document.length(),
                modTime = toUnixSeconds(document.lastModified()),
                mimeType = guessMimeType(document),
                mediaType = if (isAudioFile(displayName)) "audio" else "video",
            )
        }

        val children = document.listFiles()
            .mapNotNull { child ->
                if (!child.canRead()) {
                    return@mapNotNull null
                }
                val childName = child.name ?: return@mapNotNull null
                val childRelPath = buildRelativePath(relPath, childName)
                when {
                    child.isDirectory -> buildTreeNode(child, childName, childRelPath, source)
                    isMediaFile(childName) -> buildTreeNode(child, childName, childRelPath, source)
                    else -> null
                }
            }
            .sortedWith(compareBy<ShareTreeNode> { it.type != "dir" }.thenBy { it.name.lowercase(Locale.ROOT) })

        return ShareTreeNode(
            type = "dir",
            name = displayName,
            relPath = relPath,
            source = source,
            children = children,
        )
    }

    private fun DocumentFile.toEntry(relPath: String): ShareEntry {
        return ShareEntry(
            name = name.orEmpty(),
            relPath = relPath,
            isDirectory = isDirectory,
            size = if (isDirectory) 0L else length(),
            modTime = toUnixSeconds(lastModified()),
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

    private fun isAudioFile(name: String): Boolean {
        val extValue = name.substringAfterLast('.', missingDelimiterValue = "")
        if (extValue.isEmpty()) {
            return false
        }
        val ext = ".${extValue.lowercase(Locale.ROOT)}"
        return ext in AUDIO_EXTENSIONS
    }

    private fun streamUrl(source: String, relPath: String): String {
        val encoded = relPath.split('/')
            .filter { it.isNotEmpty() }
            .joinToString("/") { Uri.encode(it) }
        return "/stream/${Uri.encode(source)}/$encoded"
    }

    private fun toUnixSeconds(milliseconds: Long): Long {
        return if (milliseconds > 0L) milliseconds / 1000L else 0L
    }

    companion object {
        private val AUDIO_EXTENSIONS = setOf(
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
