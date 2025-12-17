// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.SharedPreferences
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.graphics.drawable.StateListDrawable
import android.location.LocationManager
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import android.text.InputType
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.widget.*
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter
import java.util.UUID

class MainActivity : AppCompatActivity() {

    private lateinit var mainLayout: LinearLayout
    private lateinit var bleHelper: BLE
    private lateinit var hotspotHelper: Hotspot
    private lateinit var scannerHelper: Scanner
    private lateinit var prefs: SharedPreferences

    private val PERM_REQ_CODE = 100
    private val BUTTON_COLOR = Color.parseColor("#828282")
    private val BUTTON_FOCUS_COLOR = Color.parseColor("#444444")
    private val BG_COLOR = Color.parseColor("#F5F5F5")

    private var currentMode = "MENU"
    private var savedSsid = ""
    private var savedPass = ""
    private var savedIp = ""

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        prefs = getSharedPreferences("taz_config", Context.MODE_PRIVATE)

        val scrollView = ScrollView(this)
        scrollView.isFillViewport = true
        scrollView.setBackgroundColor(BG_COLOR)

        mainLayout = LinearLayout(this)
        mainLayout.orientation = LinearLayout.VERTICAL
        mainLayout.gravity = Gravity.CENTER
        mainLayout.setPadding(60, 60, 60, 60)

        scrollView.addView(mainLayout)
        setContentView(scrollView)

        bleHelper = BLE(this)
        hotspotHelper = Hotspot(this)
        scannerHelper = Scanner()

        if (savedInstanceState != null) {
            currentMode = savedInstanceState.getString("mode", "MENU")
            savedSsid = savedInstanceState.getString("ssid", "")
            savedPass = savedInstanceState.getString("pass", "")
            savedIp = savedInstanceState.getString("ip", "")
        }

        if (!prefs.contains("taz_name")) {
            val rnd = UUID.randomUUID().toString().replace("-", "").take(8)
            prefs.edit().putString("taz_name", rnd).apply()
        }

        checkPermissions()
    }

    override fun onSaveInstanceState(outState: Bundle) {
        super.onSaveInstanceState(outState)
        outState.putString("mode", currentMode)
        outState.putString("ssid", savedSsid)
        outState.putString("pass", savedPass)
        outState.putString("ip", savedIp)
    }

    private fun checkPermissions() {
        val permissions = mutableListOf<String>()
        if (ActivityCompat.checkSelfPermission(this, Manifest.permission.RECORD_AUDIO) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.RECORD_AUDIO)
        if (ActivityCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.ACCESS_FINE_LOCATION)
        if (ActivityCompat.checkSelfPermission(this, Manifest.permission.CHANGE_WIFI_STATE) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.CHANGE_WIFI_STATE)

        if (Build.VERSION.SDK_INT >= 33) {
            if (ActivityCompat.checkSelfPermission(this, "android.permission.NEARBY_WIFI_DEVICES") != PackageManager.PERMISSION_GRANTED) permissions.add("android.permission.NEARBY_WIFI_DEVICES")
        }

        if (Build.VERSION.SDK_INT >= 31) {
            val needed = listOf("android.permission.BLUETOOTH_SCAN", "android.permission.BLUETOOTH_ADVERTISE", "android.permission.BLUETOOTH_CONNECT")
            needed.forEach { if (ActivityCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED) permissions.add(it) }
        } else {
            if (ActivityCompat.checkSelfPermission(this, Manifest.permission.BLUETOOTH) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.BLUETOOTH)
            if (ActivityCompat.checkSelfPermission(this, Manifest.permission.BLUETOOTH_ADMIN) != PackageManager.PERMISSION_GRANTED) permissions.add(Manifest.permission.BLUETOOTH_ADMIN)
        }

        if (permissions.isNotEmpty()) {
            ActivityCompat.requestPermissions(this, permissions.toTypedArray(), PERM_REQ_CODE)
        } else {
            startApp()
        }
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, grantResults: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        startApp()
    }

    private fun startApp() {
        mainLayout.removeAllViews()
        val loading = TextView(this)
        loading.text = "Starting Engine..."
        loading.textSize = 24f
        loading.gravity = Gravity.CENTER
        loading.setTextColor(Color.BLACK)
        mainLayout.addView(loading)

        Server.startBinaryServer(this, getBinaryArgs())
        waitForServerUp()
    }

    private fun waitForServerUp() {
        Server.fetchStatus { status ->
            runOnUiThread {
                if (status != null) {
                    if (currentMode == "HOST" && savedSsid.isNotEmpty()) {
                        updateUiForHost(savedSsid, savedPass, "http://$savedIp:35248")
                    } else {
                        showMainMenu()
                    }
                } else {
                    Handler(Looper.getMainLooper()).postDelayed({ waitForServerUp() }, 500)
                }
            }
        }
    }

    private fun showMainMenu() {
        mainLayout.removeAllViews()
        currentMode = "MENU"

        val title = TextView(this)
        title.text = "TAZ"
        title.textSize = 50f
        title.typeface = Typeface.DEFAULT_BOLD
        title.setTextColor(Color.BLACK)
        title.gravity = Gravity.CENTER
        mainLayout.addView(title)

        val subtitle = TextView(this)
        subtitle.text = "Temporary Autonomous Zone"
        subtitle.textSize = 16f
        subtitle.setTextColor(Color.DKGRAY)
        subtitle.gravity = Gravity.CENTER
        subtitle.setPadding(0, 0, 0, 80)
        mainLayout.addView(subtitle)

        val btnHost = createMenuButton("WiFi Server") { startHostMode() }
        mainLayout.addView(btnHost)

        val btnClient = createMenuButton("Radio Scan") { startClientMode() }
        mainLayout.addView(btnClient)

        val btnScan = createMenuButton("Network Scan") { startScanMode() }
        mainLayout.addView(btnScan)

        val btnSettings = createMenuButton("Settings") { showSettings() }
        mainLayout.addView(btnSettings)

        val btnLocal = createMenuButton("Open") { startLocalMode() }
        mainLayout.addView(btnLocal)
    }

    private fun createMenuButton(text: String, onClick: () -> Unit): Button {
        val btn = Button(this)
        btn.text = text
        btn.setTextColor(Color.WHITE)
        btn.textSize = 16f
        btn.setPadding(30, 30, 30, 30)
        btn.isAllCaps = false
        btn.setOnClickListener { onClick() }
        btn.isFocusable = true

        val defaultShape = GradientDrawable()
        defaultShape.shape = GradientDrawable.RECTANGLE
        defaultShape.cornerRadius = 8f
        defaultShape.setColor(BUTTON_COLOR)

        val focusedShape = GradientDrawable()
        focusedShape.shape = GradientDrawable.RECTANGLE
        focusedShape.cornerRadius = 8f
        focusedShape.setColor(BUTTON_FOCUS_COLOR)

        val states = StateListDrawable()
        states.addState(intArrayOf(android.R.attr.state_focused), focusedShape)
        states.addState(intArrayOf(), defaultShape)

        btn.background = states

        val params = LinearLayout.LayoutParams(
            LinearLayout.LayoutParams.MATCH_PARENT,
            LinearLayout.LayoutParams.WRAP_CONTENT
        )
        params.setMargins(0, 0, 0, 30)
        btn.layoutParams = params
        return btn
    }

    private fun showSettings() {
        val initialPublic = prefs.getBoolean("public_host", true)
        mainLayout.removeAllViews()

        val title = TextView(this)
        title.text = "Settings"
        title.textSize = 30f
        title.setTextColor(Color.BLACK)
        title.gravity = Gravity.CENTER
        title.setPadding(0, 0, 0, 60)
        mainLayout.addView(title)

        val checkPublic = CheckBox(this)
        checkPublic.text = "Public"
        checkPublic.textSize = 16f
        checkPublic.setTextColor(Color.BLACK)
        checkPublic.isChecked = initialPublic
        mainLayout.addView(checkPublic)

        val checkBle = CheckBox(this)
        checkBle.text = "Bluetooth"
        checkBle.textSize = 16f
        checkBle.setTextColor(Color.BLACK)
        checkBle.isChecked = prefs.getBoolean("share_ble", true)
        mainLayout.addView(checkBle)

        val nameLabel = TextView(this)
        nameLabel.text = "Name"
        nameLabel.textSize = 16f
        nameLabel.setTextColor(Color.BLACK)
        nameLabel.setPadding(10, 40, 0, 10)
        mainLayout.addView(nameLabel)

        val nameInput = EditText(this)
        nameInput.inputType = InputType.TYPE_CLASS_TEXT
        nameInput.setText(prefs.getString("taz_name", ""))
        mainLayout.addView(nameInput)

        val passLabel = TextView(this)
        passLabel.text = "Password"
        passLabel.textSize = 16f
        passLabel.setTextColor(Color.BLACK)
        passLabel.setPadding(10, 20, 0, 10)
        mainLayout.addView(passLabel)

        val passInput = EditText(this)
        passInput.hint = "Leave empty for none"
        passInput.inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD
        passInput.setText(prefs.getString("password", ""))
        mainLayout.addView(passInput)

        val spacer = View(this)
        mainLayout.addView(spacer, LinearLayout.LayoutParams(1, 60))

        val btnSave = createMenuButton("Save & Return") {
            val newPublic = checkPublic.isChecked
            prefs.edit()
                .putBoolean("public_host", newPublic)
                .putBoolean("share_ble", checkBle.isChecked)
                .putString("taz_name", nameInput.text.toString())
                .putString("password", passInput.text.toString())
                .apply()

            Server.restartBinaryServer(this, getBinaryArgs())
            showMainMenu()
        }
        mainLayout.addView(btnSave)

        val infoText = TextView(this)
        infoText.text = "Fetching info..."
        infoText.textSize = 15f
        infoText.setTextColor(Color.DKGRAY)
        infoText.setPadding(20, 20, 0, 0)
        infoText.setLineSpacing(0f, 1.2f)
        mainLayout.addView(infoText)

        Server.fetchStatus { status ->
            runOnUiThread {
                if (status != null) {
                    val sb = StringBuilder()
                    sb.append("Name: ${status.name}\n")
                    sb.append("Uptime: ${status.uptime}\n")
                    sb.append("Version: ${status.version}\n")
                    sb.append("Port: ${status.port}\n")
                    sb.append("IPs:\n    127.0.0.1\n")
                    if (checkPublic.isChecked) {
                        sb.append(hotspotHelper.getAllIps().joinToString("\n") { "    $it" })
                    }
                    infoText.text = sb.toString()
                } else {
                    infoText.text = "Server unreachable"
                }
            }
        }
    }

    private fun getBinaryArgs(): List<String> {
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

    private fun startLocalMode() {
        openWebView("http://127.0.0.1:35248/")
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

                currentMode = "HOST"
                savedSsid = ssid
                savedPass = pass
                savedIp = ip

                if (prefs.getBoolean("share_ble", true)) {
                    bleHelper.startAdvertising(creds)
                }

                updateUiForHost(ssid, pass, "http://$ip:35248")
            },
            onFailure = {
                Toast.makeText(this, "Hotspot Failed", Toast.LENGTH_SHORT).show()
                showMainMenu()
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
                        showMainMenu()
                    }
                )
            },
            onError = {
                status.text = "Scan Failed"
                Handler(Looper.getMainLooper()).postDelayed({ showMainMenu() }, 2000)
            }
        )
    }

    private fun startScanMode() {
        mainLayout.removeAllViews()

        val title = TextView(this)
        title.text = "Scanning..."
        title.textSize = 24f
        title.gravity = Gravity.CENTER
        title.setTextColor(Color.BLACK)
        title.setPadding(0, 0, 0, 30)
        mainLayout.addView(title)

        val scrollContainer = LinearLayout(this)
        scrollContainer.orientation = LinearLayout.VERTICAL
        mainLayout.addView(scrollContainer)

        val ips = hotspotHelper.getAllIps()
        var hasStarted = false

        for (ip in ips) {
            if (ip.startsWith("192.168.") || ip.startsWith("10.") || ip.startsWith("172.")) {
                hasStarted = true
                scannerHelper.scanSubnet(ip,
                    onFound = { foundIp, name ->
                        runOnUiThread {
                            val btn = createMenuButton("$foundIp\n$name") {
                                openWebView("http://$foundIp:35248")
                            }
                            scrollContainer.addView(btn)
                        }
                    },
                    onFinish = {
                        runOnUiThread {
                            title.text = "Scan Finished"
                        }
                    }
                )
                break
            }
        }

        if (!hasStarted) {
            title.text = "No private network found"
        }

        val btnBack = createMenuButton("Back") {
            showMainMenu()
        }
        val params = LinearLayout.LayoutParams(
            LinearLayout.LayoutParams.MATCH_PARENT,
            LinearLayout.LayoutParams.WRAP_CONTENT
        )
        params.setMargins(0, 50, 0, 0)
        btnBack.layoutParams = params
        mainLayout.addView(btnBack)
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
        tvTitle.setTextColor(Color.BLACK)
        tvTitle.setPadding(0, 0, 0, 30)
        mainLayout.addView(tvTitle)

        val qrImage = ImageView(this)
        val params = LinearLayout.LayoutParams(dpToPx(250f), dpToPx(250f))
        params.setMargins(0, 20, 0, 20)
        qrImage.layoutParams = params
        mainLayout.addView(qrImage)

        val tvInfo = TextView(this)
        tvInfo.gravity = Gravity.CENTER
        tvInfo.textSize = 16f
        tvInfo.setTextColor(Color.DKGRAY)
        tvInfo.setPadding(20, 20, 20, 40)
        mainLayout.addView(tvInfo)

        val btnToggle = createMenuButton("Show Browser QR") {
            isShowingWifi = !isShowingWifi
            updateUiState(tvTitle, qrImage, tvInfo, wifiBitmap, urlBitmap, isShowingWifi, ssid, pass, publicUrl)
        }
        btnToggle.setOnClickListener {
            isShowingWifi = !isShowingWifi
            updateUiState(tvTitle, qrImage, tvInfo, wifiBitmap, urlBitmap, isShowingWifi, ssid, pass, publicUrl)
            btnToggle.text = if (isShowingWifi) "Show Browser QR" else "Show WiFi QR"
        }
        mainLayout.addView(btnToggle)

        val btnOpen = createMenuButton("Open Local") {
            openWebView("http://127.0.0.1:35248/")
        }
        mainLayout.addView(btnOpen)

        updateUiState(tvTitle, qrImage, tvInfo, wifiBitmap, urlBitmap, true, ssid, pass, publicUrl)
    }

    private fun updateUiState(
        title: TextView, img: ImageView, info: TextView,
        wifiBmp: Bitmap?, urlBmp: Bitmap?, showingWifi: Boolean,
        ssid: String, pass: String, url: String
    ) {
        if (showingWifi) {
            title.text = "1. Connect WiFi"
            img.setImageBitmap(wifiBmp)
            val formattedPass = pass.chunked(4).joinToString("\u00B7")
            info.text = "SSID: $ssid\nCode: $formattedPass"
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
        if (!isChangingConfigurations) {
            bleHelper.stopAdvertising()
            hotspotHelper.stopHost()
            currentMode = "MENU"
        }
    }
}