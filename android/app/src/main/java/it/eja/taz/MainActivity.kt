// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.SharedPreferences
import android.content.pm.PackageManager
import android.location.LocationManager
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import java.util.UUID

class MainActivity : AppCompatActivity() {

    lateinit var prefs: SharedPreferences
    lateinit var bleHelper: BLE
    lateinit var hotspotHelper: Hotspot
    lateinit var ui: AppUI

    private val PERM_REQ_CODE = 100

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        prefs = getSharedPreferences("taz_config", Context.MODE_PRIVATE)
        if (!prefs.contains("taz_name")) {
            val rnd = UUID.randomUUID().toString().replace("-", "").take(8)
            prefs.edit().putString("taz_name", rnd).apply()
        }

        bleHelper = BLE(this)
        hotspotHelper = Hotspot(this)
        ui = AppUI(this)

        setContentView(ui.rootView)
        checkPermissions()
    }

    private fun checkPermissions() {
        val permissions = mutableListOf<String>()
        val needed = mutableListOf(
            Manifest.permission.RECORD_AUDIO,
            Manifest.permission.ACCESS_FINE_LOCATION,
            Manifest.permission.CHANGE_WIFI_STATE
        )

        if (Build.VERSION.SDK_INT >= 33) needed.add("android.permission.NEARBY_WIFI_DEVICES")

        if (Build.VERSION.SDK_INT >= 31) {
            needed.add("android.permission.BLUETOOTH_SCAN")
            needed.add("android.permission.BLUETOOTH_ADVERTISE")
            needed.add("android.permission.BLUETOOTH_CONNECT")
        } else {
            needed.add(Manifest.permission.BLUETOOTH)
            needed.add(Manifest.permission.BLUETOOTH_ADMIN)
        }

        needed.forEach {
            if (ActivityCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED) {
                permissions.add(it)
            }
        }

        if (permissions.isNotEmpty()) {
            ActivityCompat.requestPermissions(this, permissions.toTypedArray(), PERM_REQ_CODE)
        } else {
            startApp()
        }
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        startApp()
    }

    private fun startApp() {
        ui.showLoading("Starting Engine...")
        Server.startBinaryServer(this, getBinaryArgs())
        waitForServerUp()
    }

    private fun waitForServerUp() {
        Server.fetchStatus { status ->
            runOnUiThread {
                if (status != null) ui.showMainMenu()
                else android.os.Handler(android.os.Looper.getMainLooper())
                    .postDelayed({ waitForServerUp() }, 500)
            }
        }
    }

    fun getBinaryArgs(): List<String> {
        val args = mutableListOf<String>()
        val name = prefs.getString("taz_name", "")
        if (!name.isNullOrEmpty()) {
            args.add("--name")
            args.add(name)
        }
        if (prefs.getBoolean("public_host", true)) {
            args.add("--web-host")
            args.add("0.0.0.0")
        }
        val pass = prefs.getString("password", "")
        if (!pass.isNullOrEmpty()) {
            args.add("--password")
            args.add(pass)
        }
        return args
    }

    fun startHostMode() {
        val running = hotspotHelper.getRunningConfig()
        if (running != null) {
            ui.showHostView(running.first, running.second, "http://${running.third}:35248")
            return
        }
        ui.showLoading("Starting Hotspot...")
        hotspotHelper.startHost(
            onSuccess = { ssid, pass, ip ->
                if (prefs.getBoolean(
                        "share_ble",
                        true
                    )
                ) bleHelper.startAdvertising("$ssid\t$pass\t$ip")
                ui.showHostView(ssid, pass, "http://$ip:35248")
            },
            onFailure = {
                Toast.makeText(this, "Hotspot Failed", Toast.LENGTH_SHORT).show()
                ui.showMainMenu()
            }
        )
    }

    fun startClientMode() {
        val lm = getSystemService(Context.LOCATION_SERVICE) as LocationManager
        if (!lm.isProviderEnabled(LocationManager.GPS_PROVIDER)) {
            startActivity(Intent(Settings.ACTION_LOCATION_SOURCE_SETTINGS))
            return
        }

        ui.showLoading("Radio Scanning...") {
            bleHelper.stopScanning()
            ui.showMainMenu()
        }

        bleHelper.scanAndConnect(
            onResult = { ssid, pass, ip ->
                ui.showLoading("Connecting WiFi...")
                hotspotHelper.connectToWifi(
                    ssid, pass,
                    { ui.openWeb("http://$ip:35248") },
                    { ui.showMainMenu() }
                )
            },
            onError = { ui.showMainMenu() }
        )
    }

    override fun onDestroy() {
        super.onDestroy()
        if (!isChangingConfigurations) {
            bleHelper.stopAdvertising()
            hotspotHelper.stopHost()
        }
    }

    fun exitApp() {
        hotspotHelper.stopHost()
        bleHelper.stopAdvertising()
        Server.stopBinaryServer()
        finishAffinity()
        System.exit(0)
    }
}