// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.InetSocketAddress
import java.net.Socket
import java.net.URL
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit

class Scanner {

    fun scanSubnet(localIp: String, onFound: (String, String) -> Unit, onFinish: () -> Unit) {
        val parts = localIp.split(".")
        if (parts.size != 4) {
            onFinish()
            return
        }

        val subnet = "${parts[0]}.${parts[1]}.${parts[2]}"
        val myLast = parts[3].toIntOrNull() ?: 0

        Thread {
            val executor = Executors.newFixedThreadPool(30)

            for (i in 1..254) {
                if (i == myLast) continue
                val targetIp = "$subnet.$i"

                executor.execute {
                    checkHost(targetIp, onFound)
                }
            }

            executor.shutdown()
            try {
                executor.awaitTermination(10, TimeUnit.SECONDS)
            } catch (e: Exception) { }

            onFinish()
        }.start()
    }

    private fun checkHost(ip: String, onFound: (String, String) -> Unit) {
        try {
            val socket = Socket()
            socket.connect(InetSocketAddress(ip, 35248), 200)
            socket.close()

            val name = fetchName(ip)
            onFound(ip, name)
        } catch (e: Exception) { }
    }

    private fun fetchName(ip: String): String {
        return try {
            val url = URL("http://$ip:35248/status")
            val conn = url.openConnection() as HttpURLConnection
            conn.connectTimeout = 1000
            conn.readTimeout = 5000
            if (conn.responseCode == 200) {
                val text = conn.inputStream.bufferedReader().readText()
                val json = JSONObject(text)
                json.optString("name", "Taz Node")
            } else {
                "Taz Node"
            }
        } catch (e: Exception) {
            "Taz Node"
        }
    }
}