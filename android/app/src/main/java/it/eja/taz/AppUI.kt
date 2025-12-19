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

    private val LEAF_GREEN = Color.parseColor("#7CB342")
    private val ACCENT_CYAN = Color.parseColor("#4A9B8E")
    private val WARM_ORANGE = Color.parseColor("#E8A843")
    private val LIGHT_CREAM = Color.parseColor("#F4F1DE")
    private val RUST_RED = Color.parseColor("#A0522D")

    init {
        rootView.isFillViewport = true
        rootView.setBackgroundColor(LIGHT_CREAM)
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
            setTextColor(LEAF_GREEN)
            typeface = Typeface.create(Typeface.DEFAULT, Typeface.BOLD)
            setPadding(0, 0, 0, 50)
        }
        mainLayout.addView(tv)

        if (onBack != null) {
            mainLayout.addView(createBtn("Back", ACCENT_CYAN) {
                onBack()
            })
        }
    }

    fun showMainMenu() {
        mainLayout.removeAllViews()
        val title = TextView(act).apply {
            text = "TAZ"
            textSize = 56f
            typeface = Typeface.DEFAULT_BOLD
            gravity = Gravity.CENTER
            setTextColor(LEAF_GREEN)
        }
        val sub = TextView(act).apply {
            text = "Temporary Autonomous Zone"
            textSize = 14f
            gravity = Gravity.CENTER
            setPadding(0, 10, 0, 100)
            setTextColor(ACCENT_CYAN)
        }
        mainLayout.addView(title)
        mainLayout.addView(sub)

        mainLayout.addView(createBtn("Wi-Fi Server", ACCENT_CYAN) { act.startHostMode() })

        val row1 = LinearLayout(act).apply {
            orientation = LinearLayout.HORIZONTAL
            layoutParams = LinearLayout.LayoutParams(-1, -2)
        }

        val btnBle = createBtn("Bluetooth", ACCENT_CYAN) { act.startClientMode() }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(0, 0, 10, 20) }
        }
        val btnNet = createBtn("Network", ACCENT_CYAN) { showScanMode() }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(10, 0, 0, 20) }
        }

        row1.addView(btnBle)
        row1.addView(btnNet)
        mainLayout.addView(row1)

        mainLayout.addView(View(act).apply {
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, 50)
        })

        mainLayout.addView(
            createBtn(
                "Local Node",
                LEAF_GREEN
            ) { openWeb("http://127.0.0.1:35248/") })

        val row2 = LinearLayout(act).apply {
            orientation = LinearLayout.HORIZONTAL
            layoutParams = LinearLayout.LayoutParams(-1, -2)
        }

        val btnSettings = createBtn("Settings", WARM_ORANGE) { showSettings() }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(0, 0, 10, 20) }
        }
        val btnExit = createBtn("Exit", RUST_RED) { act.exitApp() }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(10, 0, 0, 20) }
        }

        row2.addView(btnSettings)
        row2.addView(btnExit)
        mainLayout.addView(row2)
    }

    fun showScanMode() {
        mainLayout.removeAllViews()
        val title = TextView(act).apply {
            text = "Network Scanning..."
            textSize = 24f
            gravity = Gravity.CENTER
            setPadding(0, 0, 0, 50)
            setTextColor(LEAF_GREEN)
            typeface = Typeface.create(Typeface.DEFAULT, Typeface.BOLD)
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
                                container.addView(createBtn("${p.name}\n${p.ip}", LEAF_GREEN) {
                                    openWeb("http://${p.ip}:35248")
                                })
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
        mainLayout.addView(createBtn("Back", ACCENT_CYAN) {
            handler.removeCallbacksAndMessages(null)
            showMainMenu()
        })
    }

    fun showHostView(ssid: String, pass: String, url: String) {
        mainLayout.removeAllViews()
        var showingWifi = true

        val tv = TextView(act).apply {
            textSize = 22f
            gravity = Gravity.CENTER
            setPadding(0, 0, 0, 30)
            setTextColor(LEAF_GREEN)
            typeface = Typeface.create(Typeface.DEFAULT, Typeface.BOLD)
        }
        val img = ImageView(act).apply {
            val size = (act.resources.displayMetrics.density * 250).toInt()
            layoutParams = LinearLayout.LayoutParams(size, size)
        }
        val info = TextView(act).apply {
            gravity = Gravity.CENTER
            setPadding(0, 20, 0, 40)
            setTextColor(ACCENT_CYAN)
            textSize = 14f
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

        mainLayout.addView(tv)
        mainLayout.addView(img)
        mainLayout.addView(info)
        mainLayout.addView(createBtn("Switch QR Mode", ACCENT_CYAN) {
            showingWifi = !showingWifi
            update()
        })

        val actionRow = LinearLayout(act).apply {
            orientation = LinearLayout.HORIZONTAL
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            )
        }

        val btnStop = createBtn("Stop", WARM_ORANGE) {
            act.hotspotHelper.stopHost()
            act.bleHelper.stopAdvertising()
            showMainMenu()
        }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(0, 0, 10, 20) }
        }

        val btnBack = createBtn("Back", LEAF_GREEN) {
            showMainMenu()
        }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(10, 0, 0, 20) }
        }

        actionRow.addView(btnStop)
        actionRow.addView(btnBack)
        mainLayout.addView(actionRow)

        update()
    }

    fun showSettings() {
        mainLayout.removeAllViews()
        mainLayout.addView(TextView(act).apply {
            text = "Settings"
            textSize = 36f
            setTextColor(LEAF_GREEN)
            typeface = Typeface.DEFAULT_BOLD
            gravity = Gravity.CENTER
            setPadding(0, 0, 0, 60)
        })

        val nameInput = EditText(act).apply {
            setText(act.prefs.getString("taz_name", ""))
            setTextColor(LEAF_GREEN)
            setHintTextColor(ACCENT_CYAN)
            background = createInputBackground()
            setPadding(30, 30, 30, 30)
        }

        val passInput = EditText(act).apply {
            hint = "Password"
            inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD
            setText(act.prefs.getString("password", ""))
            setTextColor(LEAF_GREEN)
            setHintTextColor(ACCENT_CYAN)
            background = createInputBackground()
            setPadding(30, 30, 30, 30)
        }

        val checkRow = LinearLayout(act).apply {
            orientation = LinearLayout.HORIZONTAL
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { setMargins(0, 30, 0, 20) }
        }

        val pubCheck = CheckBox(act).apply {
            text = "Public"
            isChecked = act.prefs.getBoolean("public_host", true)
            setTextColor(LEAF_GREEN)
            textSize = 16f
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
        }

        val bleCheck = CheckBox(act).apply {
            text = "Bluetooth"
            isChecked = act.prefs.getBoolean("share_ble", true)
            setTextColor(LEAF_GREEN)
            textSize = 16f
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
        }
        checkRow.addView(pubCheck)
        checkRow.addView(bleCheck)

        mainLayout.addView(TextView(act).apply {
            text = "Name"
            setTextColor(LEAF_GREEN)
            textSize = 14f
            typeface = Typeface.DEFAULT_BOLD
            setPadding(0, 0, 0, 10)
        })
        mainLayout.addView(nameInput)
        mainLayout.addView(TextView(act).apply {
            text = "Password"
            setTextColor(LEAF_GREEN)
            textSize = 14f
            typeface = Typeface.DEFAULT_BOLD
            setPadding(0, 30, 0, 10)
        })
        mainLayout.addView(passInput)
        mainLayout.addView(checkRow)

        val btnRow = LinearLayout(act).apply {
            orientation = LinearLayout.HORIZONTAL
            layoutParams = LinearLayout.LayoutParams(-1, -2).apply { setMargins(0, 40, 0, 0) }
        }

        val saveBtn = createBtn("Save", LEAF_GREEN) {
            act.prefs.edit().putString("taz_name", nameInput.text.toString())
                .putString("password", passInput.text.toString())
                .putBoolean("public_host", pubCheck.isChecked)
                .putBoolean("share_ble", bleCheck.isChecked).apply()
            Server.restartBinaryServer(act, act.getBinaryArgs())
            showMainMenu()
        }.apply {
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f).apply { setMargins(0, 0, 10, 0) }
        }

        val backBtn = createBtn("Back", ACCENT_CYAN) { showMainMenu() }.apply {
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

    private fun createBtn(txt: String, bgColor: Int, onClick: () -> Unit): Button {
        return Button(act).apply {
            text = txt
            setTextColor(Color.WHITE)
            textSize = 16f
            isAllCaps = false
            typeface = Typeface.create(Typeface.DEFAULT, Typeface.BOLD)

            val shape = GradientDrawable().apply {
                cornerRadius = 20f
                setColor(bgColor)
            }
            background = shape
            elevation = 8f
            stateListAnimator = null

            setOnClickListener { onClick() }

            val params = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            )
            params.setMargins(0, 0, 0, 20)
            layoutParams = params
            setPadding(40, 40, 40, 40)
        }
    }

    private fun createInputBackground(): GradientDrawable {
        return GradientDrawable().apply {
            cornerRadius = 16f
            setColor(Color.WHITE)
            setStroke(3, ACCENT_CYAN)
        }
    }

    private fun genQR(content: String): Bitmap? {
        return try {
            val matrix = QRCodeWriter().encode(content, BarcodeFormat.QR_CODE, 512, 512)
            val bmp = Bitmap.createBitmap(512, 512, Bitmap.Config.RGB_565)

            for (x in 0 until 512) {
                for (y in 0 until 512) {
                    bmp.setPixel(x, y, if (matrix.get(x, y)) LEAF_GREEN else Color.WHITE)
                }
            }

            val rounded = Bitmap.createBitmap(512, 512, Bitmap.Config.ARGB_8888)
            val canvas = Canvas(rounded)
            val paint = Paint(Paint.ANTI_ALIAS_FLAG)
            val rect = RectF(0f, 0f, 512f, 512f)

            canvas.drawRoundRect(rect, 40f, 40f, paint.apply {
                shader = BitmapShader(bmp, Shader.TileMode.CLAMP, Shader.TileMode.CLAMP)
            })

            rounded
        } catch (e: Exception) {
            null
        }
    }
}