// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.R
import android.content.Context
import android.os.ParcelFileDescriptor
import org.json.JSONObject
import java.io.BufferedReader
import java.io.File
import java.io.FileInputStream
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL

data class ServerStatus(
    val name: String,
    val version: String,
    val uptime: Long,
    val port: Int
)

object Server {

    private var isStarted = false
    private var currentPid = 0
    private const val STATUS_URL = "http://127.0.0.1:35248/status"

    fun setupFilesFolder(context: Context) {
        try {
            val sourceApk = File(context.applicationInfo.sourceDir)
            val baseFiles = File(context.filesDir, "files")
            val apkDir = File(baseFiles, "sys")
            if (!apkDir.exists()) apkDir.mkdirs()
            val destFile = File(apkDir, "taz.apk")
            if (!destFile.exists()) {
                sourceApk.inputStream().use { input ->
                    destFile.outputStream().use { output ->
                        input.copyTo(output)
                    }
                }
            }
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    fun startBinaryServer(context: Context, extraArgs: List<String>) {
        if (isStarted) return
        isStarted = true

        Thread {
            try {
                setupFilesFolder(context)
                val libDir = context.applicationInfo.nativeLibraryDir
                val binPath = "$libDir/libtaz.so"
                val appDir = context.filesDir.absolutePath
                if (!File(binPath).exists()) return@Thread

                val args = extraArgs.toTypedArray()
                val env = arrayOf("HOME=$appDir", "TMPDIR=${context.cacheDir.absolutePath}")
                val pid = IntArray(1)

                val fd = Native.createSubprocess(binPath, appDir, args, env, pid)
                if (fd > 0) {
                    currentPid = pid[0]
                    val pfd = ParcelFileDescriptor.adoptFd(fd)
                    val input = FileInputStream(pfd.fileDescriptor)
                    val reader = BufferedReader(InputStreamReader(input))
                    while (reader.readLine() != null) { }
                }
            } catch (e: Exception) {
                e.printStackTrace()
            } finally {
                isStarted = false
                currentPid = 0
            }
        }.start()
    }

    fun restartBinaryServer(context: Context, extraArgs: List<String>) {
        if (currentPid > 0) {
            android.os.Process.killProcess(currentPid)
            currentPid = 0
            isStarted = false
        }
        Thread.sleep(200) 
        startBinaryServer(context, extraArgs)
    }

    fun fetchStatus(callback: (ServerStatus?) -> Unit) {
        Thread {
            try {
                val url = URL(STATUS_URL)
                val conn = url.openConnection() as HttpURLConnection
                conn.connectTimeout = 1000
                conn.readTimeout = 1000
                conn.requestMethod = "GET"

                if (conn.responseCode == 200) {
                    val text = conn.inputStream.bufferedReader().readText()
                    val json = JSONObject(text)
                    
                    val status = ServerStatus(
                        name = json.optString("name", "Unknown"),
                        version = json.optString("version", "Unknown"),
                        uptime = json.optLong("uptime", 0),
                        port = json.optInt("port",0)
                    )
                    callback(status)
                    return@Thread
                }
            } catch (e: Exception) { }
            callback(null)
        }.start()
    }
}
