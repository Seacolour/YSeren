package io.github.seacolour.yseren.mobile.share

import org.json.JSONObject
import org.junit.Assert.assertTrue
import org.junit.Test

class TreeContractTest {
    @Test
    fun androidSerializerMatchesSharedTreeFixture() {
        val expected = loadFixture("tree-response.v1.json")
        val root = parseNode(expected.getJSONObject("root"))
        val actual = TreeContractJson.response(
            root = root,
            generatedAt = expected.getLong("generatedAt"),
        )

        assertTrue(
            "Serialized tree did not match shared fixture.\nExpected: $expected\nActual: $actual",
            expected.similar(actual),
        )
    }

    @Test
    fun defaultAndroidTreeWrapsTheSourceUnderRoot() {
        val sourceRoot = ShareTreeNode(
            type = "dir",
            name = ANDROID_SOURCE,
            relPath = "",
            source = ANDROID_SOURCE,
            children = listOf(
                ShareTreeNode(
                    type = "file",
                    name = "movie.mp4",
                    relPath = "movie.mp4",
                    source = ANDROID_SOURCE,
                    url = "/stream/android/movie.mp4",
                    size = 10L,
                    modTime = 1_700_000_000L,
                    mediaType = "video",
                ),
            ),
        )
        val root = ShareTreeNode(
            type = "dir",
            name = "root",
            relPath = "",
            children = listOf(sourceRoot),
        )

        val actual = TreeContractJson.response(root, generatedAt = 1_700_000_001L)
        val serializedRoot = actual.getJSONObject("root")
        val serializedSource = serializedRoot.getJSONArray("children").getJSONObject(0)
        val serializedFile = serializedSource.getJSONArray("children").getJSONObject(0)

        assertTrue(!serializedRoot.has("source"))
        assertTrue(serializedSource.getString("source") == ANDROID_SOURCE)
        assertTrue(serializedFile.getString("url") == "/stream/android/movie.mp4")
        assertTrue(serializedFile.getLong("modTime") == 1_700_000_000L)
    }

    private fun parseNode(json: JSONObject): ShareTreeNode {
        val childrenJson = json.optJSONArray("children")
        val children = if (childrenJson == null) {
            emptyList()
        } else {
            (0 until childrenJson.length()).map { parseNode(childrenJson.getJSONObject(it)) }
        }
        return ShareTreeNode(
            type = json.getString("type"),
            name = json.getString("name"),
            relPath = json.getString("relPath"),
            source = json.optString("source"),
            url = json.optString("url").takeIf { it.isNotEmpty() },
            size = json.optLong("size"),
            modTime = json.optLong("modTime"),
            mimeType = json.optString("mimeType").takeIf { it.isNotEmpty() },
            mediaType = json.optString("mediaType").takeIf { it.isNotEmpty() },
            children = children,
        )
    }
}
