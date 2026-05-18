package io.github.seacolour.yseren.mobile.share

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import io.github.seacolour.yseren.mobile.AppPrefs
import io.github.seacolour.yseren.mobile.MainActivity
import io.github.seacolour.yseren.mobile.R

class ShareServerService : Service() {
    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        return when (intent?.action) {
            ACTION_STOP -> {
                ShareServerController.stop()
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
                START_NOT_STICKY
            }

            ACTION_START, null -> {
                val config = AppPrefs(this).loadConfig()
                if (config == null) {
                    ShareServerController.stop()
                    stopSelf()
                    return START_NOT_STICKY
                }

                createNotificationChannel()
                val startResult = ShareServerController.start(this, config)
                if (startResult.isFailure) {
                    stopSelf()
                    return START_NOT_STICKY
                }

                startForeground(NOTIFICATION_ID, buildNotification(config.displayName, config.port))
                START_STICKY
            }

            else -> START_NOT_STICKY
        }
    }

    override fun onDestroy() {
        ShareServerController.stop()
        super.onDestroy()
    }

    private fun buildNotification(displayName: String, port: Int): Notification {
        val openIntent = PendingIntent.getActivity(
            this,
            1001,
            Intent(this, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
            },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )

        val stopIntent = PendingIntent.getService(
            this,
            1002,
            Intent(this, ShareServerService::class.java).setAction(ACTION_STOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )

        val contentText = "正在共享：$displayName · 端口 $port"
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(getString(R.string.notification_title))
            .setContentText(contentText)
            .setStyle(NotificationCompat.BigTextStyle().bigText(contentText))
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setContentIntent(openIntent)
            .addAction(0, getString(R.string.stop_share), stopIntent)
            .build()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return
        }
        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        val channel = NotificationChannel(
            CHANNEL_ID,
            getString(R.string.notification_channel_name),
            NotificationManager.IMPORTANCE_LOW,
        )
        channel.description = getString(R.string.notification_channel_description)
        manager.createNotificationChannel(channel)
    }

    companion object {
        const val ACTION_START = "io.github.seacolour.yseren.mobile.action.START"
        const val ACTION_STOP = "io.github.seacolour.yseren.mobile.action.STOP"

        private const val CHANNEL_ID = "yseren_android_share"
        private const val NOTIFICATION_ID = 1479
    }
}
