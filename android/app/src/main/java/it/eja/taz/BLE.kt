// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package it.eja.taz

import android.annotation.SuppressLint
import android.bluetooth.*
import android.bluetooth.le.*
import android.content.Context
import android.os.Handler
import android.os.Looper
import android.os.ParcelUuid
import java.util.UUID

@SuppressLint("MissingPermission")
class BLE(private val context: Context) {

    private val SERVICE_UUID = UUID.fromString("82935248-0000-1000-8000-00805f9b34fb")
    private val CHAR_UUID = UUID.fromString("82935248-0000-1000-8000-00805f9b34fc")

    private val btManager = context.getSystemService(Context.BLUETOOTH_SERVICE) as BluetoothManager
    private val adapter = btManager.adapter

    private var gattServer: BluetoothGattServer? = null
    private var advertiser: BluetoothLeAdvertiser? = null
    private var hostCredentials = ""

    fun startAdvertising(credentials: String) {
        hostCredentials = credentials

        gattServer = btManager.openGattServer(context, object : BluetoothGattServerCallback() {
            override fun onCharacteristicReadRequest(
                device: BluetoothDevice?, requestId: Int, offset: Int,
                characteristic: BluetoothGattCharacteristic?
            ) {
                if (characteristic?.uuid == CHAR_UUID) {
                    val fullData = hostCredentials.toByteArray(Charsets.UTF_8)
                    if (offset < fullData.size) {
                        val chunk = fullData.copyOfRange(offset, fullData.size)
                        gattServer?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, offset, chunk)
                    } else {
                        gattServer?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, offset, ByteArray(0))
                    }
                } else {
                    gattServer?.sendResponse(device, requestId, BluetoothGatt.GATT_FAILURE, 0, null)
                }
            }
        })

        val service = BluetoothGattService(SERVICE_UUID, BluetoothGattService.SERVICE_TYPE_PRIMARY)
        val characteristic = BluetoothGattCharacteristic(
            CHAR_UUID,
            BluetoothGattCharacteristic.PROPERTY_READ,
            BluetoothGattCharacteristic.PERMISSION_READ
        )
        service.addCharacteristic(characteristic)
        gattServer?.addService(service)

        advertiser = adapter.bluetoothLeAdvertiser
        val settings = AdvertiseSettings.Builder()
            .setAdvertiseMode(AdvertiseSettings.ADVERTISE_MODE_LOW_LATENCY)
            .setConnectable(true)
            .build()
        val data = AdvertiseData.Builder()
            .setIncludeDeviceName(false)
            .addServiceUuid(ParcelUuid(SERVICE_UUID))
            .build()

        advertiser?.startAdvertising(settings, data, object : AdvertiseCallback() {})
    }

    fun stopAdvertising() {
        try {
            advertiser?.stopAdvertising(object : AdvertiseCallback() {})
            gattServer?.close()
        } catch (e: Exception) {}
    }

    fun scanAndConnect(onResult: (ssid: String, pass: String, ip: String) -> Unit, onError: () -> Unit) {
        val scanner = adapter.bluetoothLeScanner
        if (scanner == null) {
            onError()
            return
        }

        val settings = ScanSettings.Builder().setScanMode(ScanSettings.SCAN_MODE_BALANCED).build()

        val callback = object : ScanCallback() {
            override fun onScanResult(callbackType: Int, result: ScanResult?) {
                val device = result?.device
                val record = result?.scanRecord
                if (device != null && record != null) {
                    if (record.serviceUuids?.contains(ParcelUuid(SERVICE_UUID)) == true) {
                        try {
                            scanner.stopScan(this)
                        } catch (e: Exception) {}
                        connectGatt(device, onResult)
                    }
                }
            }
            override fun onScanFailed(errorCode: Int) {
                onError()
            }
        }
        scanner.startScan(emptyList(), settings, callback)
    }

    private fun connectGatt(device: BluetoothDevice, onResult: (String, String, String) -> Unit) {
        device.connectGatt(context, false, object : BluetoothGattCallback() {
            override fun onConnectionStateChange(gatt: BluetoothGatt?, status: Int, newState: Int) {
                if (newState == BluetoothProfile.STATE_CONNECTED) {
                    gatt?.requestMtu(512)
                }
            }

            override fun onMtuChanged(gatt: BluetoothGatt?, mtu: Int, status: Int) {
                gatt?.discoverServices()
            }

            override fun onServicesDiscovered(gatt: BluetoothGatt?, status: Int) {
                val service = gatt?.getService(SERVICE_UUID)
                val charac = service?.getCharacteristic(CHAR_UUID)
                if (charac != null) {
                    gatt?.readCharacteristic(charac)
                }
            }

            override fun onCharacteristicRead(gatt: BluetoothGatt?, charac: BluetoothGattCharacteristic?, status: Int) {
                if (status == BluetoothGatt.GATT_SUCCESS && charac != null) {
                    val data = String(charac.value, Charsets.UTF_8)
                    val parts = data.split("\t")
                    if (parts.size >= 3) {
                        gatt?.disconnect()
                        gatt?.close()
                        Handler(Looper.getMainLooper()).post {
                            onResult(parts[0], parts[1], parts[2])
                        }
                    }
                }
            }
        })
    }
}
