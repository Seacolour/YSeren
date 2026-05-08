package io.github.seacolour.yseren.mobile

import android.content.Context
import android.net.Uri

data class ShareConfig(
    val treeUri: Uri,
    val displayName: String,
    val port: Int,
)

class AppPrefs(context: Context) {
    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    fun saveTree(uri: Uri, displayName: String) {
        prefs.edit()
            .putString(KEY_TREE_URI, uri.toString())
            .putString(KEY_TREE_NAME, displayName)
            .apply()
    }

    fun savePort(port: Int) {
        prefs.edit()
            .putInt(KEY_PORT, port)
            .apply()
    }

    fun clearTree() {
        prefs.edit()
            .remove(KEY_TREE_URI)
            .remove(KEY_TREE_NAME)
            .apply()
    }

    fun loadConfig(): ShareConfig? {
        val uriString = prefs.getString(KEY_TREE_URI, null) ?: return null
        return ShareConfig(
            treeUri = Uri.parse(uriString),
            displayName = prefs.getString(KEY_TREE_NAME, DEFAULT_TREE_NAME) ?: DEFAULT_TREE_NAME,
            port = prefs.getInt(KEY_PORT, DEFAULT_PORT),
        )
    }

    companion object {
        private const val PREFS_NAME = "yseren_android_share"
        private const val KEY_TREE_URI = "tree_uri"
        private const val KEY_TREE_NAME = "tree_name"
        private const val KEY_PORT = "port"

        const val DEFAULT_PORT = 1479
        private const val DEFAULT_TREE_NAME = "Shared Media"
    }
}
