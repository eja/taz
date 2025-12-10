// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.location.LocationManager
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import android.util.TypedValue
import android.view.Gravity
import android.widget.Button
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter

class MainActivity : AppCompatActivity() {

    private lateinit var mainLayout: LinearLayout
    private lateinit var bleHelper: BLE
    private lateinit var hotspotHelper: Hotspot
    private val PERM_REQ_CODE = 100
    private val BUTTON_COLOR = Color.parseColor("#808080")

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        mainLayout = LinearLayout(this)
        mainLayout.orientation = LinearLayout.VERTICAL
        mainLayout.gravity = Gravity.CENTER
        mainLayout.setPadding(60, 60, 60, 60)
        mainLayout.setBackgroundColor(Color.parseColor("#FAFAFA"))
        setContentView(mainLayout)

        bleHelper = BLE(this)
        hotspotHelper = Hotspot(this)

        checkPermissions()
    }

    private fun checkPermissions() {
        val permissions = mutableListOf<String>()
        if (ActivityCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.ACCESS_FINE_LOCATION)
        if (ActivityCompat.checkSelfPermission(this, Manifest.permission.CHANGE_WIFI_STATE) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.CHANGE_WIFI_STATE)

        if (Build.VERSION.SDK_INT >= 33) {
            if (ActivityCompat.checkSelfPermission(this, "android.permission.NEARBY_WIFI_DEVICES") != PackageManager.PERMISSION_GRANTED) permissions.add("android.permission.NEARBY_WIFI_DEVICES")
        }

        if (Build.VERSION.SDK_INT >= 31) {
            val needed = listOf(
                "android.permission.BLUETOOTH_SCAN",
                "android.permission.BLUETOOTH_ADVERTISE",
                "android.permission.BLUETOOTH_CONNECT"
            )
            needed.forEach {
                if (ActivityCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED) permissions.add(it)
            }
        } else {
            if (ActivityCompat.checkSelfPermission(this, Manifest.permission.BLUETOOTH) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.BLUETOOTH)
            if (ActivityCompat.checkSelfPermission(this, Manifest.permission.BLUETOOTH_ADMIN) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.BLUETOOTH_ADMIN)
        }

        if (permissions.isNotEmpty()) {
            ActivityCompat.requestPermissions(this, permissions.toTypedArray(), PERM_REQ_CODE)
        } else {
            showModeDialog()
        }
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, grantResults: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        showModeDialog()
    }

    private fun showModeDialog() {
        mainLayout.removeAllViews()

        val title = TextView(this)
        title.text = "TAZ"
        title.textSize = 40f
        title.typeface = Typeface.DEFAULT_BOLD
        title.setTextColor(Color.parseColor("#263238"))
        title.gravity = Gravity.CENTER
        title.setPadding(0, 0, 0, 20)
        mainLayout.addView(title)

        val subtitle = TextView(this)
        subtitle.text = "Temporary Autonomous Zone"
        subtitle.textSize = 16f
        subtitle.setTextColor(Color.GRAY)
        subtitle.gravity = Gravity.CENTER
        subtitle.setPadding(0, 0, 0, 80)
        mainLayout.addView(subtitle)

        val btnHost = createStyledButton("Start Server") {
            startHostMode()
        }
        mainLayout.addView(btnHost)

        val btnClient = createStyledButton("Connect Client") {
            startClientMode()
        }
        mainLayout.addView(btnClient)

        val btnLocal = createStyledButton("Standalone Mode") {
            startLocalMode()
        }
        mainLayout.addView(btnLocal)
    }

    private fun createStyledButton(text: String, onClick: () -> Unit): Button {
        val btn = Button(this)
        btn.text = text
        btn.setTextColor(Color.WHITE)
        btn.textSize = 16f
        btn.setPadding(40, 40, 40, 40)
        btn.isAllCaps = false
        btn.setOnClickListener { onClick() }

        val shape = GradientDrawable()
        shape.shape = GradientDrawable.RECTANGLE
        shape.cornerRadius = 20f
        shape.setColor(BUTTON_COLOR)
        btn.background = shape

        val params = LinearLayout.LayoutParams(
            LinearLayout.LayoutParams.MATCH_PARENT,
            LinearLayout.LayoutParams.WRAP_CONTENT
        )
        params.setMargins(0, 0, 0, 30)
        btn.layoutParams = params
        btn.elevation = 8f

        return btn
    }

    private fun startLocalMode() {
        Server.startBinaryServer(this)
        Handler(Looper.getMainLooper()).postDelayed({
            openWebView("http://127.0.0.1:35248/")
        }, 1000)
    }

    private fun startHostMode() {
        mainLayout.removeAllViews()
        val status = TextView(this)
        status.text = "Starting Hotspot..."
        status.textSize = 20f
        status.gravity = Gravity.CENTER
        mainLayout.addView(status)

        hotspotHelper.startHost(
            onSuccess = { ssid, pass, ip ->
                val creds = "$ssid\t$pass\t$ip"
                Server.startBinaryServer(this@MainActivity)
                bleHelper.startAdvertising(creds)
                updateUiForHost(ssid, pass, "http://$ip:35248")
            },
            onFailure = {
                Toast.makeText(this, "Hotspot Failed", Toast.LENGTH_SHORT).show()
                startLocalMode()
            }
        )
    }

    private fun startClientMode() {
        val lm = getSystemService(Context.LOCATION_SERVICE) as LocationManager
        if (!lm.isProviderEnabled(LocationManager.GPS_PROVIDER) && !lm.isProviderEnabled(LocationManager.NETWORK_PROVIDER)) {
            Toast.makeText(this, "Enable Location for BLE", Toast.LENGTH_LONG).show()
            startActivity(Intent(Settings.ACTION_LOCATION_SOURCE_SETTINGS))
            return
        }

        mainLayout.removeAllViews()
        val status = TextView(this)
        status.text = "Scanning for Host..."
        status.textSize = 24f
        status.gravity = Gravity.CENTER
        mainLayout.addView(status)

        bleHelper.scanAndConnect(
            onResult = { ssid, pass, ip ->
                status.text = "Found. Connecting WiFi..."
                hotspotHelper.connectToWifi(
                    ssid, pass,
                    onSuccess = { openWebView("http://$ip:35248") },
                    onFailure = {
                        Toast.makeText(this, "WiFi Failed", Toast.LENGTH_SHORT).show()
                        startLocalMode()
                    }
                )
            },
            onError = {
                status.text = "Scan Failed"
            }
        )
    }

    private fun updateUiForHost(ssid: String, pass: String, publicUrl: String) {
        mainLayout.removeAllViews()
        val wifiQrContent = "WIFI:T:WPA;S:$ssid;P:$pass;;"
        val wifiBitmap = generateQrBitmap(wifiQrContent)
        val urlBitmap = generateQrBitmap(publicUrl)
        var isShowingWifi = true

        val tvTitle = TextView(this)
        tvTitle.textSize = 22f
        tvTitle.gravity = Gravity.CENTER
        tvTitle.setTypeface(null, Typeface.BOLD)
        tvTitle.setTextColor(Color.parseColor("#263238"))
        tvTitle.setPadding(0, 0, 0, 30)
        mainLayout.addView(tvTitle)

        val qrImage = ImageView(this)
        val params = LinearLayout.LayoutParams(
            dpToPx(250f), dpToPx(250f)
        )
        params.setMargins(0, 20, 0, 20)
        qrImage.layoutParams = params
        mainLayout.addView(qrImage)

        val tvInfo = TextView(this)
        tvInfo.gravity = Gravity.CENTER
        tvInfo.textSize = 16f
        tvInfo.setTextColor(Color.DKGRAY)
        tvInfo.setPadding(20, 20, 20, 40)
        mainLayout.addView(tvInfo)

        val btnToggle = createStyledButton("Show Browser QR") {
            isShowingWifi = !isShowingWifi
            updateUiState(tvTitle, qrImage, tvInfo, wifiBitmap, urlBitmap, isShowingWifi, ssid, pass, publicUrl)
        }

        btnToggle.setOnClickListener {
            isShowingWifi = !isShowingWifi
            updateUiState(tvTitle, qrImage, tvInfo, wifiBitmap, urlBitmap, isShowingWifi, ssid, pass, publicUrl)
            btnToggle.text = if (isShowingWifi) "Show Browser QR" else "Show WiFi QR"
        }

        mainLayout.addView(btnToggle)

        val btnOpen = createStyledButton("Open Local") {
            openWebView(publicUrl)
        }
        mainLayout.addView(btnOpen)

        tvTitle.text = "1. Connect WiFi"
        qrImage.setImageBitmap(wifiBitmap)
        tvInfo.text = "SSID: $ssid\nPassword: $pass"
        btnToggle.text = "Show Browser QR"
    }

    private fun updateUiState(
        title: TextView, img: ImageView, info: TextView,
        wifiBmp: Bitmap?, urlBmp: Bitmap?, showingWifi: Boolean,
        ssid: String, pass: String, url: String
    ) {
        if (showingWifi) {
            title.text = "1. Connect WiFi"
            img.setImageBitmap(wifiBmp)
            info.text = "SSID: $ssid\nPassword: $pass"
        } else {
            title.text = "2. Open Browser"
            img.setImageBitmap(urlBmp)
            info.text = url
        }
    }

    private fun generateQrBitmap(content: String): Bitmap? {
        return try {
            val writer = QRCodeWriter()
            val bitMatrix = writer.encode(content, BarcodeFormat.QR_CODE, 512, 512)
            val w = bitMatrix.width
            val h = bitMatrix.height
            val bmp = Bitmap.createBitmap(w, h, Bitmap.Config.RGB_565)
            for (x in 0 until w) {
                for (y in 0 until h) {
                    bmp.setPixel(x, y, if (bitMatrix[x, y]) Color.BLACK else Color.WHITE)
                }
            }
            bmp
        } catch (e: Exception) { null }
    }

    private fun openWebView(url: String) {
        val intent = Intent(this, WebActivity::class.java)
        intent.putExtra("url", url)
        startActivity(intent)
    }

    private fun dpToPx(dp: Float): Int {
        return TypedValue.applyDimension(TypedValue.COMPLEX_UNIT_DIP, dp, resources.displayMetrics).toInt()
    }

    override fun onDestroy() {
        super.onDestroy()
        bleHelper.stopAdvertising()
        hotspotHelper.stopHost()
    }
}
