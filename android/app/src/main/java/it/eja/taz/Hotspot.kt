// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.net.wifi.WifiConfiguration
import android.net.wifi.WifiManager
import android.net.wifi.WifiNetworkSpecifier
import android.os.Build
import android.os.Handler
import android.os.Looper
import java.net.Inet4Address
import java.net.NetworkInterface
import java.util.Collections

class Hotspot(private val context: Context) {

    private var reservation: WifiManager.LocalOnlyHotspotReservation? = null

    fun getAllIps(): Set<String> {
        val ips = mutableSetOf<String>()
        try {
            val interfaces = Collections.list(NetworkInterface.getNetworkInterfaces())
            for (intf in interfaces) {
                if (intf.isLoopback || !intf.isUp) continue
                val addrs = Collections.list(intf.inetAddresses)
                for (addr in addrs) {
                    if (addr is Inet4Address && !addr.isLoopbackAddress) {
                        addr.hostAddress?.let { ips.add(it) }
                    }
                }
            }
        } catch (e: Exception) { }
        return ips
    }

    fun startHost(onSuccess: (ssid: String, pass: String, ip: String) -> Unit, onFailure: () -> Unit) {
        val wifiManager = context.getSystemService(Context.WIFI_SERVICE) as WifiManager
        val preIps = getAllIps()

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            try {
                wifiManager.startLocalOnlyHotspot(object : WifiManager.LocalOnlyHotspotCallback() {
                    override fun onStarted(res: WifiManager.LocalOnlyHotspotReservation) {
                        super.onStarted(res)
                        reservation = res
                        val config = res.wifiConfiguration
                        val ssid = removeQuotes(config?.SSID ?: "Unknown")
                        val pass = removeQuotes(config?.preSharedKey ?: "Unknown")

                        val postIps = getAllIps()
                        val diff = postIps.minus(preIps)
                        val ip = diff.firstOrNull()

                        if (ip != null) {
                            onSuccess(ssid, pass, ip)
                        } else {
                            onFailure()
                        }
                    }
                    override fun onFailed(reason: Int) {
                        super.onFailed(reason)
                        onFailure()
                    }
                }, Handler(Looper.getMainLooper()))
            } catch (e: Exception) {
                onFailure()
            }
        } else {
            onFailure()
        }
    }

    fun stopHost() {
        try {
            reservation?.close()
        } catch (e: Exception) {}
    }

    fun connectToWifi(ssid: String, pass: String, onSuccess: () -> Unit, onFailure: () -> Unit) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            try {
                val cleanSsid = removeQuotes(ssid)
                val cleanPass = removeQuotes(pass)

                val specifier = WifiNetworkSpecifier.Builder()
                    .setSsid(cleanSsid)
                    .setWpa2Passphrase(cleanPass)
                    .build()

                val request = NetworkRequest.Builder()
                    .addTransportType(NetworkCapabilities.TRANSPORT_WIFI)
                    .removeCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                    .setNetworkSpecifier(specifier)
                    .build()

                val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
                cm.requestNetwork(request, object : ConnectivityManager.NetworkCallback() {
                    override fun onAvailable(network: Network) {
                        super.onAvailable(network)
                        cm.bindProcessToNetwork(network)
                        onSuccess()
                    }
                    override fun onUnavailable() {
                        super.onUnavailable()
                        connectLegacyWifi(ssid, pass, onSuccess, onFailure)
                    }
                })
            } catch (e: Exception) {
                connectLegacyWifi(ssid, pass, onSuccess, onFailure)
            }
        } else {
            connectLegacyWifi(ssid, pass, onSuccess, onFailure)
        }
    }

    private fun connectLegacyWifi(ssid: String, pass: String, onSuccess: () -> Unit, onFailure: () -> Unit) {
        try {
            val wifiManager = context.getSystemService(Context.WIFI_SERVICE) as WifiManager
            val wifiConfig = WifiConfiguration()
            wifiConfig.SSID = String.format("\"%s\"", removeQuotes(ssid))
            wifiConfig.preSharedKey = String.format("\"%s\"", removeQuotes(pass))

            val netId = wifiManager.addNetwork(wifiConfig)
            if (netId == -1) {
                onFailure()
                return
            }
            wifiManager.disconnect()
            wifiManager.enableNetwork(netId, true)
            wifiManager.reconnect()

            Handler(Looper.getMainLooper()).postDelayed({
                onSuccess()
            }, 5000)
        } catch (e: Exception) {
            onFailure()
        }
    }

    private fun removeQuotes(text: String): String {
        if (text.length > 1 && text.startsWith("\"") && text.endsWith("\"")) {
            return text.substring(1, text.length - 1)
        }
        return text
    }
}
