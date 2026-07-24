package io.github.seacolour.yseren.mobile.share

import org.json.JSONArray
import org.json.JSONObject

internal const val ANDROID_SOURCE = "android"

data class ShareTreeNode(
    val type: String,
    val name: String,
    val relPath: String,
    val source: String = "",
    val url: String? = null,
    val size: Long = 0L,
    val modTime: Long = 0L,
    val mimeType: String? = null,
    val mediaType: String? = null,
    val children: List<ShareTreeNode> = emptyList(),
)

internal object TreeContractJson {
    fun response(
        root: ShareTreeNode,
        generatedAt: Long,
        source: String = "",
    ): JSONObject {
        return JSONObject()
            .put("generatedAt", generatedAt)
            .apply {
                if (source.isNotBlank()) {
                    put("source", source)
                }
            }
            .put("root", node(root))
    }

    fun node(value: ShareTreeNode): JSONObject {
        return JSONObject()
            .put("type", value.type)
            .put("name", value.name)
            .put("relPath", value.relPath)
            .apply {
                if (value.source.isNotBlank()) {
                    put("source", value.source)
                }
                if (value.type == "file") {
                    value.url?.let { put("url", it) }
                    put("size", value.size)
                    put("modTime", value.modTime)
                    value.mimeType?.let { put("mimeType", it) }
                    value.mediaType?.let { put("mediaType", it) }
                }
                if (value.children.isNotEmpty()) {
                    put(
                        "children",
                        JSONArray().apply {
                            value.children.forEach { put(node(it)) }
                        },
                    )
                }
            }
    }
}
