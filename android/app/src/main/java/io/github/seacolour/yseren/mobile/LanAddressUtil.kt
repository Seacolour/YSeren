package io.github.seacolour.yseren.mobile

import java.net.NetworkInterface

object LanAddressUtil {
    fun listLanIpv4(): List<String> {
        val results = linkedSetOf<String>()
        val interfaces = NetworkInterface.getNetworkInterfaces() ?: return emptyList()

        while (interfaces.hasMoreElements()) {
            val network = interfaces.nextElement()
            if (!network.isUp || network.isLoopback) {
                continue
            }

            val addresses = network.inetAddresses
            while (addresses.hasMoreElements()) {
                val address = addresses.nextElement()
                val ip = address.hostAddress ?: continue
                if (address.isLoopbackAddress || ip.contains(':')) {
                    continue
                }
                if (ip.startsWith("169.254.")) {
                    continue
                }
                if (!isPrivateIpv4(ip)) {
                    continue
                }
                results += ip
            }
        }

        return results.toList().sorted()
    }

    private fun isPrivateIpv4(ip: String): Boolean {
        return ip.startsWith("10.") ||
            ip.startsWith("192.168.") ||
            ip.startsWith("172.16.") ||
            ip.startsWith("172.17.") ||
            ip.startsWith("172.18.") ||
            ip.startsWith("172.19.") ||
            ip.startsWith("172.2") ||
            ip.startsWith("172.30.") ||
            ip.startsWith("172.31.")
    }
}
