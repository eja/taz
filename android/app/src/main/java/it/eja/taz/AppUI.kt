// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.content.Intent
import android.graphics.*
import android.graphics.drawable.*
import android.os.*
import android.text.InputType
import android.view.*
import android.widget.*
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter

class AppUI(private val act: MainActivity) {

    val rootView: ScrollView = ScrollView(act)
    private val mainLayout = LinearLayout(act)

    private val BUTTON_COLOR = Color.parseColor("#828282")
    private val BG_COLOR = Color.parseColor("#F5F5F5")

    init {
        rootView.isFillViewport = true
        rootView.setBackgroundColor(BG_COLOR)
        mainLayout.orientation = LinearLayout.VERTICAL
        mainLayout.gravity = Gravity.CENTER
        mainLayout.setPadding(60, 60, 60, 60)
        rootView.addView(mainLayout)
    }

    fun showLoading(msg: String, onBack: (() -> Unit)? = null) {
        mainLayout.removeAllViews()

        val tv = TextView(act).apply {
            text = msg
            textSize = 24f
            gravity = Gravity.CENTER
            setTextColor(Color.BLACK)
            setPadding(0, 0, 0, 50)
        }
        mainLayout.addView(tv)

        if (onBack != null) {
            mainLayout.addView(createBtn("Back") {
                onBack()
            })
        }
    }

    fun showMainMenu() {
        mainLayout.removeAllViews()
        val title = TextView(act).apply {
            text = "TAZ"
            textSize = 50f
            typeface = Typeface.DEFAULT_BOLD
            gravity = Gravity.CENTER
            setTextColor(Color.BLACK)
        }
        val sub = TextView(act).apply {
            text = "Temporary Autonomous Zone"
            textSize = 14f
            gravity = Gravity.CENTER
            setPadding(0, 0, 0, 80)
            setTextColor(Color.DKGRAY)
        }
        mainLayout.addView(title)
        mainLayout.addView(sub)

        mainLayout.addView(createBtn("WiFi Server") { act.startHostMode() })
        mainLayout.addView(createBtn("Radio Scan") { act.startClientMode() })
        mainLayout.addView(createBtn("Network Scan") { showScanMode() })
        mainLayout.addView(createBtn("Settings") { showSettings() })
        mainLayout.addView(createBtn("Open") { openWeb("http://127.0.0.1:35248/") })
        mainLayout.addView(createBtn("Exit") { act.exitApp() })
    }

    fun showScanMode() {
        mainLayout.removeAllViews()
        val title = TextView(act).apply {
            text = "Network Scanning..."; textSize = 24f; gravity = Gravity.CENTER; setPadding(
            0, 0, 0, 50
        ); setTextColor(Color.BLACK)
        }
        val container = LinearLayout(act).apply { orientation = LinearLayout.VERTICAL }

        mainLayout.addView(title)
        mainLayout.addView(container)

        val handler = Handler(Looper.getMainLooper())
        var timeLeft = 15
        val myName = "TAZ-" + act.prefs.getString("taz_name", "")

        val task = object : Runnable {
            override fun run() {
                Server.fetchStatus { s ->
                    act.runOnUiThread {
                        val peers = s?.discovery?.filter { it.name != myName } ?: emptyList()
                        if (peers.isNotEmpty()) {
                            title.text = "Available Nodes"
                            container.removeAllViews()
                            peers.forEach { p ->
                                container.addView(createBtn("${p.name}\n${p.ip}") { openWeb("http://${p.ip}:35248") })
                            }
                        } else if (timeLeft > 0) {
                            timeLeft -= 2
                            handler.postDelayed(this, 2000)
                        } else {
                            title.text = "No Available Nodes"
                        }
                    }
                }
            }
        }
        handler.post(task)
        mainLayout.addView(createBtn("Back") {
            handler.removeCallbacksAndMessages(null)
            showMainMenu()
        })
    }

    fun showHostView(ssid: String, pass: String, url: String) {
        mainLayout.removeAllViews()
        var showingWifi = true

        val tv = TextView(act).apply {
            textSize = 20f; gravity = Gravity.CENTER; setPadding(
            0, 0, 0, 30
        ); setTextColor(Color.BLACK)
        }
        val img = ImageView(act).apply {
            val size = (act.resources.displayMetrics.density * 250).toInt()
            layoutParams = LinearLayout.LayoutParams(size, size)
        }
        val info = TextView(act).apply {
            gravity = Gravity.CENTER; setPadding(0, 20, 0, 40); setTextColor(Color.DKGRAY)
        }

        fun update() {
            if (showingWifi) {
                tv.text = "1. Connect WiFi"
                img.setImageBitmap(genQR("WIFI:S:$ssid;P:$pass;;"))
                info.text = "SSID: $ssid\nPass: $pass"
            } else {
                tv.text = "2. Open Browser"
                img.setImageBitmap(genQR(url))
                info.text = url
            }
        }

        mainLayout.addView(tv); mainLayout.addView(img); mainLayout.addView(info)
        mainLayout.addView(createBtn("Switch QR Mode") {
            showingWifi = !showingWifi; update()
        })
        mainLayout.addView(createBtn("Stop Server") {
            act.hotspotHelper.stopHost()
            act.bleHelper.stopAdvertising()
            showMainMenu()
        })
        mainLayout.addView(createBtn("Back") {
            showMainMenu()
        })

        update()
    }

    fun showSettings() {
        mainLayout.removeAllViews()
        mainLayout.addView(TextView(act).apply {
            text = "Settings"; textSize = 32f;
            setTextColor(Color.BLACK); gravity = Gravity.CENTER; setPadding(0, 0, 0, 60)
        })
        val nameInput = EditText(act).apply { setText(act.prefs.getString("taz_name", "")) }
        val passInput = EditText(act).apply {
            hint = "Password"
            inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD
            setText(act.prefs.getString("password", ""))
        }
        val pubCheck = CheckBox(act).apply {
            text = "Public"
            isChecked = act.prefs.getBoolean("public_host", true)
            setTextColor(Color.BLACK)
        }
        val bleCheck = CheckBox(act).apply {
            text = "Bluetooth"; isChecked = act.prefs.getBoolean("share_ble", true); setTextColor(
            Color.BLACK
        )
        }

        mainLayout.addView(TextView(act).apply { text = "Name"; setTextColor(Color.BLACK) })
        mainLayout.addView(nameInput)
        mainLayout.addView(TextView(act).apply { text = "Password"; setTextColor(Color.BLACK) })
        mainLayout.addView(passInput)
        mainLayout.addView(pubCheck)
        mainLayout.addView(bleCheck)

        val btnRow = LinearLayout(act).apply {
            orientation = LinearLayout.HORIZONTAL
            layoutParams = LinearLayout.LayoutParams(-1, -2).apply { setMargins(0, 40, 0, 0) }
        }
        val saveBtn = createBtn("Save") {
            act.prefs.edit().putString("taz_name", nameInput.text.toString())
                .putString("password", passInput.text.toString())
                .putBoolean("public_host", pubCheck.isChecked)
                .putBoolean("share_ble", bleCheck.isChecked).apply()
            Server.restartBinaryServer(act, act.getBinaryArgs())
            showMainMenu()
        }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(0, 0, 10, 0) }
        }

        val backBtn = createBtn("Back") { showMainMenu() }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(10, 0, 0, 0) }
        }

        btnRow.addView(saveBtn)
        btnRow.addView(backBtn)
        mainLayout.addView(btnRow)
    }

    fun openWeb(url: String) {
        val intent = Intent(act, WebActivity::class.java).apply { putExtra("url", url) }
        act.startActivity(intent)
    }

    private fun createBtn(txt: String, onClick: () -> Unit): Button {
        return Button(act).apply {
            text = txt
            setTextColor(Color.WHITE)
            isAllCaps = false
            val shape = GradientDrawable().apply {
                cornerRadius = 8f
                setColor(BUTTON_COLOR)
            }
            background = shape
            setOnClickListener { onClick() }
            val params = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT
            )
            params.setMargins(0, 0, 0, 30)
            layoutParams = params
        }
    }

    private fun genQR(content: String): Bitmap? {
        return try {
            val matrix = QRCodeWriter().encode(content, BarcodeFormat.QR_CODE, 512, 512)
            val bmp = Bitmap.createBitmap(512, 512, Bitmap.Config.RGB_565)
            for (x in 0 until 512) {
                for (y in 0 until 512) {
                    bmp.setPixel(x, y, if (matrix.get(x, y)) Color.BLACK else Color.WHITE)
                }
            }
            bmp
        } catch (e: Exception) {
            null
        }
    }
}