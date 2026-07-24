package io.github.seacolour.yseren.mobile.share

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class HttpRangeTest {
    @Test
    fun sharedRangeFixtureMatchesAndroidParser() {
        val fixture = loadFixture("range-cases.v1.json")
        val totalLength = fixture.getJSONObject("resource").getString("content").length.toLong()
        val cases = fixture.getJSONArray("cases")

        for (index in 0 until cases.length()) {
            val case = cases.getJSONObject(index)
            val header = case.getString("range").ifBlank { null }
            val selection = HttpRange.parse(header, totalLength)
            when (case.getInt("status")) {
                200 -> assertEquals(case.getString("name"), RangeSelection.Full, selection)
                206 -> {
                    assertTrue(case.getString("name"), selection is RangeSelection.Partial)
                    selection as RangeSelection.Partial
                    val expected = parseContentRange(case.getString("contentRange"))
                    assertEquals(case.getString("name"), expected.first, selection.start)
                    assertEquals(case.getString("name"), expected.second, selection.end)
                    assertEquals(case.getString("contentLength").toLong(), selection.length)
                }

                416 -> assertEquals(case.getString("name"), RangeSelection.Unsatisfiable, selection)
                else -> error("Unsupported fixture status ${case.getInt("status")}")
            }
        }
    }

    @Test
    fun malformedAndMultipleRangesAreUnsatisfiable() {
        val invalidHeaders = listOf(
            "items=0-3",
            "bytes=",
            "bytes=-0",
            "bytes=4-2",
            "bytes=0-1,4-5",
            "bytes=abc-def",
        )

        invalidHeaders.forEach { header ->
            assertEquals(header, RangeSelection.Unsatisfiable, HttpRange.parse(header, 10L))
        }
    }

    @Test
    fun rangeAgainstEmptyResourceIsUnsatisfiable() {
        assertEquals(RangeSelection.Full, HttpRange.parse(null, 0L))
        assertEquals(RangeSelection.Unsatisfiable, HttpRange.parse("bytes=0-", 0L))
    }

    private fun parseContentRange(value: String): Pair<Long, Long> {
        val range = value.removePrefix("bytes ").substringBefore('/')
        val parts = range.split('-', limit = 2)
        return parts[0].toLong() to parts[1].toLong()
    }
}

internal fun loadFixture(name: String): JSONObject {
    val stream = checkNotNull(HttpRangeTest::class.java.classLoader?.getResourceAsStream(name)) {
        "Fixture not found: $name"
    }
    return stream.bufferedReader().use { JSONObject(it.readText()) }
}
