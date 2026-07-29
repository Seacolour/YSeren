package io.github.seacolour.yseren.mobile.share

internal sealed interface RangeSelection {
    data object Full : RangeSelection

    data class Partial(
        val start: Long,
        val end: Long,
    ) : RangeSelection {
        val length: Long
            get() = (end - start) + 1
    }

    data object Unsatisfiable : RangeSelection
}

internal object HttpRange {
    fun parse(header: String?, totalLength: Long): RangeSelection {
        if (header.isNullOrBlank()) {
            return RangeSelection.Full
        }
        if (totalLength <= 0L) {
            return RangeSelection.Unsatisfiable
        }

        val trimmed = header.trim()
        if (!trimmed.startsWith("bytes=") || ',' in trimmed) {
            return RangeSelection.Unsatisfiable
        }

        val candidate = trimmed.removePrefix("bytes=").trim()
        val parts = candidate.split('-', limit = 2)
        if (parts.size != 2) {
            return RangeSelection.Unsatisfiable
        }

        val startPart = parts[0].trim()
        val endPart = parts[1].trim()
        if (startPart.isEmpty()) {
            val suffixLength = endPart.toLongOrNull()
                ?.takeIf { it > 0L }
                ?: return RangeSelection.Unsatisfiable
            val start = (totalLength - suffixLength).coerceAtLeast(0L)
            return RangeSelection.Partial(start, totalLength - 1)
        }

        val start = startPart.toLongOrNull()
            ?.takeIf { it >= 0L && it < totalLength }
            ?: return RangeSelection.Unsatisfiable
        val end = if (endPart.isEmpty()) {
            totalLength - 1
        } else {
            endPart.toLongOrNull()
                ?.takeIf { it >= start }
                ?.coerceAtMost(totalLength - 1)
                ?: return RangeSelection.Unsatisfiable
        }
        return RangeSelection.Partial(start, end)
    }
}
