// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.content.Context
import android.os.ParcelFileDescriptor
import org.json.JSONObject
import java.io.BufferedReader
import java.io.File
import java.io.FileInputStream
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL

data class Peer(
    val ip: String,
    val name: String,
    val version: String
)

data class ServerStatus(
    val name: String,
    val version: String,
    val uptime: Long,
    val port: Int,
    val discovery: List<Peer>
)

object Server {

    private var isStarted = false
    private var currentPid = 0
    private const val STATUS_URL = "http://127.0.0.1:35248/status"

    init {
        System.loadLibrary("launcher")
    }

    @JvmStatic
    private external fun createSubprocess(
        cmd: String,
        cwd: String,
        args: Array<String>,
        envVars: Array<String>,
        processIdArray: IntArray
    ): Int

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

                val fd = createSubprocess(binPath, appDir, args, env, pid)
                if (fd > 0) {
                    currentPid = pid[0]
                    val pfd = ParcelFileDescriptor.adoptFd(fd)
                    val input = FileInputStream(pfd.fileDescriptor)
                    val reader = BufferedReader(InputStreamReader(input))
                    while (reader.readLine() != null) {
                    }
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
                conn.readTimeout = 1500
                conn.requestMethod = "GET"

                if (conn.responseCode == 200) {
                    val text = conn.inputStream.bufferedReader().readText()
                    val json = JSONObject(text)

                    val peerList = mutableListOf<Peer>()
                    val discoveryArr = json.optJSONArray("discovery")
                    if (discoveryArr != null) {
                        for (i in 0 until discoveryArr.length()) {
                            val p = discoveryArr.getJSONObject(i)
                            peerList.add(
                                Peer(
                                    ip = p.optString("ip"),
                                    name = p.optString("name"),
                                    version = p.optString("version")
                                )
                            )
                        }
                    }

                    val status = ServerStatus(
                        name = json.optString("name", "Unknown"),
                        version = json.optString("version", "Unknown"),
                        uptime = json.optLong("uptime", 0),
                        port = json.optInt("port", 0),
                        discovery = peerList
                    )
                    callback(status)
                    return@Thread
                }
            } catch (e: Exception) {
            }
            callback(null)
        }.start()
    }

    fun stopBinaryServer() {
        if (currentPid > 0) {
            android.os.Process.killProcess(currentPid)
            currentPid = 0
            isStarted = false
        }
    }
}