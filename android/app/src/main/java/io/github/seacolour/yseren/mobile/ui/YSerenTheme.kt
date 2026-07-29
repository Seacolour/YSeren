package io.github.seacolour.yseren.mobile.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

internal val BrandPurple = Color(0xFF625BFF)
internal val BrandPurpleDark = Color(0xFFCBC2FF)
internal val BrandPurpleSoft = Color(0xFFECE9FF)
internal val SuccessGreen = Color(0xFF168563)
internal val ErrorRed = Color(0xFFBA1A1A)

private val LightColors = lightColorScheme(
    primary = BrandPurple,
    onPrimary = Color.White,
    primaryContainer = BrandPurpleSoft,
    onPrimaryContainer = Color(0xFF211A5A),
    secondary = Color(0xFF5E5D72),
    onSecondary = Color.White,
    secondaryContainer = Color(0xFFE5E1F4),
    onSecondaryContainer = Color(0xFF1B1A2C),
    background = Color(0xFFF7F7FC),
    onBackground = Color(0xFF1B1B21),
    surface = Color(0xFFFFFFFF),
    onSurface = Color(0xFF1B1B21),
    surfaceVariant = Color(0xFFF0EFF6),
    onSurfaceVariant = Color(0xFF5F5E6A),
    outline = Color(0xFFCAC8D3),
    outlineVariant = Color(0xFFE2E0E9),
    error = ErrorRed,
)

private val DarkColors = darkColorScheme(
    primary = BrandPurpleDark,
    onPrimary = Color(0xFF312A78),
    primaryContainer = Color(0xFF474091),
    onPrimaryContainer = Color(0xFFE7E0FF),
    secondary = Color(0xFFC8C4DA),
    background = Color(0xFF121217),
    onBackground = Color(0xFFE5E1E8),
    surface = Color(0xFF1B1B21),
    onSurface = Color(0xFFE5E1E8),
    surfaceVariant = Color(0xFF29282F),
    onSurfaceVariant = Color(0xFFC9C5D0),
    outline = Color(0xFF918F9A),
    outlineVariant = Color(0xFF44434B),
    error = Color(0xFFFFB4AB),
)

private val BaseTypography = Typography()
private val AppTypography = Typography(
    headlineSmall = TextStyle(
        fontSize = 25.sp,
        lineHeight = 31.sp,
        fontWeight = FontWeight.Bold,
    ),
    titleLarge = BaseTypography.titleLarge.copy(fontWeight = FontWeight.Bold),
    titleMedium = BaseTypography.titleMedium.copy(fontWeight = FontWeight.SemiBold),
    labelLarge = BaseTypography.labelLarge.copy(fontWeight = FontWeight.SemiBold),
)

@Composable
internal fun YSerenTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = if (isSystemInDarkTheme()) DarkColors else LightColors,
        typography = AppTypography,
        content = content,
    )
}
