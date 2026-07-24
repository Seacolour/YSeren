package io.github.seacolour.yseren.mobile

internal data class MediaScanUiState(
    val sourceKey: String? = null,
    val scanning: Boolean = false,
    val completed: Boolean = false,
    val videoCount: Int = 0,
    val audioCount: Int = 0,
    val error: String? = null,
) {
    val totalCount: Int
        get() = videoCount + audioCount
}

internal data class ShareUiState(
    val config: ShareConfig? = null,
    val running: Boolean = false,
    val localUrl: String? = null,
    val lanUrls: List<String> = emptyList(),
    val lastError: String? = null,
    val hasTreePermission: Boolean = false,
    val sourceLocation: String = "",
    val mediaScan: MediaScanUiState = MediaScanUiState(),
    val appVersion: String = "dev",
) {
    val status: ShareStatus
        get() = when {
            config == null -> ShareStatus.NoFolder
            !hasTreePermission -> ShareStatus.PermissionLost
            running -> ShareStatus.Running
            lastError != null -> ShareStatus.Error
            else -> ShareStatus.Ready
        }
}

internal enum class ShareStatus {
    NoFolder,
    Ready,
    Running,
    PermissionLost,
    Error,
}
