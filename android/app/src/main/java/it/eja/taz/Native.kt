// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

object Native {
    init {
        System.loadLibrary("launcher")
    }

    @JvmStatic
    external fun createSubprocess(
        cmd: String,
        cwd: String,
        args: Array<String>,
        envVars: Array<String>,
        processIdArray: IntArray
    ): Int
}
