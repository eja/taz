// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.content.Context
import android.os.ParcelFileDescriptor
import java.io.BufferedReader
import java.io.File
import java.io.FileInputStream
import java.io.InputStreamReader

object Server {

    fun setupFilesFolder(context: Context) {
        try {
            val sourceApk = File(context.applicationInfo.sourceDir)
            val baseFiles = File(context.filesDir, "files")
            val apkDir = File(baseFiles, "apk")
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

    fun startBinaryServer(context: Context) {
        Thread {
            try {
                setupFilesFolder(context)
                val libDir = context.applicationInfo.nativeLibraryDir
                val binPath = "$libDir/libtaz.so"
                val appDir = context.filesDir.absolutePath
                if (!File(binPath).exists()) return@Thread

                val args = arrayOf("--bbs", "bbs.jsonl", "--web-host", "0.0.0.0")
                val env = arrayOf("HOME=$appDir", "TMPDIR=${context.cacheDir.absolutePath}")
                val pid = IntArray(1)

                val fd = Native.createSubprocess(binPath, appDir, args, env, pid)
                if (fd > 0) {
                    val pfd = ParcelFileDescriptor.adoptFd(fd)
                    val input = FileInputStream(pfd.fileDescriptor)
                    val reader = BufferedReader(InputStreamReader(input))
                    while (reader.readLine() != null) { }
                }
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }.start()
    }
}
