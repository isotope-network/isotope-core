package com.example.iso_mobile

import android.Manifest
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothDevice
import android.bluetooth.BluetoothGatt
import android.bluetooth.BluetoothGattCharacteristic
import android.bluetooth.BluetoothGattServer
import android.bluetooth.BluetoothGattServerCallback
import android.bluetooth.BluetoothGattService
import android.bluetooth.BluetoothManager
import android.bluetooth.le.AdvertiseCallback
import android.bluetooth.le.AdvertiseData
import android.bluetooth.le.AdvertiseSettings
import android.bluetooth.le.BluetoothLeAdvertiser
import android.content.Context
import android.content.pm.PackageManager
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.os.Build
import android.os.ParcelUuid
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel
import java.util.UUID
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

// Импорты для gomobile
import mobile.Mobile

class MainActivity : FlutterActivity() {
    private val METHOD_CHANNEL = "isotope/nsd"
    private val EVENT_CHANNEL = "isotope/nsd/events"
    private val BLE_METHOD_CHANNEL = "isotope/ble"
    private val LIBP2P_METHOD_CHANNEL = "isotope/libp2p"
    private val MESSAGES_EVENT_CHANNEL = "isotope/messages"

    private var nsdManager: NsdManager? = null
    private var registrationListener: NsdManager.RegistrationListener? = null
    private var discoveryListener: NsdManager.DiscoveryListener? = null
    private var eventSink: EventChannel.EventSink? = null
    private var messageEventSink: EventChannel.EventSink? = null
    private var isDiscovering = false

    // BLE
    private var bluetoothAdapter: BluetoothAdapter? = null
    private var bleAdvertiser: BluetoothLeAdvertiser? = null
    private var advertiseCallback: AdvertiseCallback? = null
    private var gattServer: BluetoothGattServer? = null

    // ISOTOPE BLE UUID
    private val ISOTOPE_SERVICE_UUID: UUID = UUID.fromString("6e400001-b5a3-f393-e0a9-e50e24dcca9e")
    private val ISOTOPE_CHAR_UUID: UUID = UUID.fromString("6e400002-b5a3-f393-e0a9-e50e24dcca9e")

    // Текущий PeerID и multiaddr libp2p узла
    private var currentPeerId: String = ""
    private var currentMultiaddr: String = ""
    private var currentPort: Int = 8081

    // Для сохранения журнала
    private var saveLogResult: MethodChannel.Result? = null
    private var saveLogText: String? = null
    private val SAVE_LOG_REQUEST_CODE = 1001

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        nsdManager = getSystemService(Context.NSD_SERVICE) as NsdManager

        // ВАЖНО: Устанавливаем filesDir для gomobile ДО обработки MethodChannel
        Mobile.setFilesDir(filesDir.absolutePath)

        // NSD MethodChannel
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, METHOD_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "announce" -> {
                        val serviceName = call.argument<String>("serviceName") ?: "ISOTOPE"
                        val ip = call.argument<String>("ip") ?: ""
                        val peerId = call.argument<String>("peerId") ?: ""
                        val multiaddr = call.argument<String>("multiaddr") ?: ""
                        val port = when (val p = call.argument<Any>("port")) {
                            is Int -> p
                            is Long -> p.toInt()
                            else -> 8081
                        }
                        currentPeerId = peerId
                        currentMultiaddr = multiaddr
                        currentPort = port
                        announceService(serviceName, ip, peerId, multiaddr, port, result)
                    }
                    "stopAnnounce" -> {
                        stopAnnounce(result)
                    }
                    else -> result.notImplemented()
                }
            }

        // NSD EventChannel
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
                    eventSink = events
                    startDiscovery()
                }

                override fun onCancel(arguments: Any?) {
                    stopDiscovery()
                    eventSink = null
                }
            })

        // BLE MethodChannel
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, BLE_METHOD_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "startAdvertising" -> {
                        val deviceName = call.argument<String>("deviceName") ?: "ISOTOPE"
                        startBleAdvertising(deviceName, result)
                    }
                    "stopAdvertising" -> {
                        stopBleAdvertising(result)
                    }
                    "startGattServer" -> {
                        startGattServer(result)
                    }
                    "getOwnMac" -> {
                        val mac = bluetoothAdapter?.address ?: ""
                        result.success(mac)
                    }
                    else -> result.notImplemented()
                }
            }

        // LibP2P MethodChannel (gomobile)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, LIBP2P_METHOD_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "start" -> {
                        val ethHash = call.argument<String>("ethHash") ?: ""
                        val bootstrapPeers = call.argument<String>("bootstrapPeers") ?: ""
                        val enableMDNS = call.argument<Boolean>("enableMDNS") ?: false
                        Thread {
                            val response = Mobile.start(ethHash, bootstrapPeers, enableMDNS)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "send" -> {
                        val text = call.argument<String>("text") ?: ""
                        val ttl = when (val t = call.argument<Any>("ttl")) {
                            is Long -> t
                            is Int -> t.toLong()
                            else -> 0L
                        }
                        Thread {
                            val response = Mobile.sendMessage(text, ttl)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getMessages" -> {
                        Thread {
                            val response = Mobile.getMessages()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getPeers" -> {
                        Thread {
                            val response = Mobile.getPeers()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getStatus" -> {
                        Thread {
                            val response = Mobile.getStatus()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getMultiaddrs" -> {
                        Thread {
                            val response = Mobile.getMultiaddrs()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "connectToPeer" -> {
                        val multiaddr = call.argument<String>("multiaddr") ?: ""
                        Thread {
                            val response = Mobile.connectToPeer(multiaddr)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getFilesDir" -> {
                        result.success(filesDir.absolutePath)
                    }
                    "getLogs" -> {
                        result.success(Mobile.getLogs())
                    }
                    "saveLog" -> {
                        val text = call.argument<String>("text") ?: ""
                        saveLogText = text
                        saveLogResult = result

                        val timestamp = SimpleDateFormat("yyyyMMdd-HHmmss", Locale.US).format(Date())
                        val fileName = "isotope_log_$timestamp.txt"

                        val intent = android.content.Intent(android.content.Intent.ACTION_CREATE_DOCUMENT).apply {
                            addCategory(android.content.Intent.CATEGORY_OPENABLE)
                            type = "text/plain"
                            putExtra(android.content.Intent.EXTRA_TITLE, fileName)
                        }
                        startActivityForResult(intent, SAVE_LOG_REQUEST_CODE)
                    }
                    "stop" -> {
                        Thread {
                            val response = Mobile.stop()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "joinDHT" -> {
                        val bootstrapPeers = call.argument<String>("bootstrapPeers") ?: ""
                        Thread {
                            val response = Mobile.joinDHT(bootstrapPeers)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "findPeer" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        Thread {
                            val response = Mobile.findPeer(peerID)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "findPeersViaNetwork" -> {
                        Thread {
                            val response = Mobile.findPeersViaNetwork()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "provide" -> {
                        Thread {
                            val response = Mobile.provide()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getDHTInfo" -> {
                        Thread {
                            val response = Mobile.getDHTInfo()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    else -> result.notImplemented()
                }
            }

        // EventChannel для новых сообщений
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, MESSAGES_EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
                    messageEventSink = events
                    // Устанавливаем колбэк для сообщений
                    Mobile.setMessageCallback(object : mobile.MessageCallback {
                        override fun onMessage(message: String) {
                            runOnUiThread {
                                runCatching { messageEventSink?.success(message) }
                            }
                        }
                    })
                }

                override fun onCancel(arguments: Any?) {
                    Mobile.setMessageCallback(null)
                    messageEventSink = null
                }
            })

        // Инициализация Bluetooth
        val bluetoothManager = getSystemService(Context.BLUETOOTH_SERVICE) as BluetoothManager
        bluetoothAdapter = bluetoothManager.adapter
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: android.content.Intent?) {
        super.onActivityResult(requestCode, resultCode, data)

        if (requestCode == SAVE_LOG_REQUEST_CODE) {
            if (resultCode == android.app.Activity.RESULT_OK && data != null) {
                val uri = data.data
                if (uri != null && saveLogText != null) {
                    try {
                        contentResolver.openOutputStream(uri)?.use { outputStream ->
                            outputStream.write(saveLogText!!.toByteArray())
                        }
                        saveLogResult?.success("Сохранено")
                    } catch (e: Exception) {
                        saveLogResult?.error("SAVE_ERROR", e.message, null)
                    }
                } else {
                    saveLogResult?.success(null)
                }
            } else {
                saveLogResult?.success(null)
            }
            saveLogResult = null
            saveLogText = null
        }
    }

    private fun sendEvent(data: Map<String, Any>) {
        runOnUiThread {
            runCatching { eventSink?.success(data) }
        }
    }

    private fun announceService(
        serviceName: String,
        ip: String,
        peerId: String,
        multiaddr: String,
        port: Int,
        result: MethodChannel.Result
    ) {
        try {
            registrationListener?.let {
                runCatching { nsdManager?.unregisterService(it) }
            }

            val serviceInfo = NsdServiceInfo().apply {
                this.serviceName = serviceName
                serviceType = "_isotope._tcp."
                this.port = port
                setAttribute("ip", ip)
                if (peerId.isNotEmpty()) {
                    setAttribute("peerid", peerId)
                }
                if (multiaddr.isNotEmpty()) {
                    setAttribute("multiaddr", multiaddr)
                }
            }

            registrationListener = object : NsdManager.RegistrationListener {
                override fun onServiceRegistered(serviceInfo: NsdServiceInfo) {
                    runOnUiThread {
                        runCatching { result.success("registered") }
                    }
                }

                override fun onRegistrationFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {
                    runOnUiThread {
                        runCatching { result.error("REGISTER_FAILED", "Error code: $errorCode", null) }
                    }
                }

                override fun onServiceUnregistered(serviceInfo: NsdServiceInfo) {}

                override fun onUnregistrationFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {}
            }

            nsdManager?.registerService(serviceInfo, NsdManager.PROTOCOL_DNS_SD, registrationListener)
        } catch (e: Exception) {
            runOnUiThread {
                runCatching { result.error("ANNOUNCE_ERROR", e.message, null) }
            }
        }
    }

    private fun startDiscovery() {
        try {
            discoveryListener?.let {
                runCatching { nsdManager?.stopServiceDiscovery(it) }
            }

            isDiscovering = true

            discoveryListener = object : NsdManager.DiscoveryListener {
                override fun onDiscoveryStarted(serviceType: String) {
                    sendEvent(mapOf("event" to "started"))
                }

                override fun onServiceFound(serviceInfo: NsdServiceInfo) {
                    val resolveListener = object : NsdManager.ResolveListener {
                        override fun onResolveFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {}

                        override fun onServiceResolved(serviceInfo: NsdServiceInfo) {
                            val ipBytes = serviceInfo.attributes["ip"]
                            if (ipBytes != null) {
                                val ip = String(ipBytes, Charsets.UTF_8)
                                val address = "$ip:${serviceInfo.port}"

                                val peerIdBytes = serviceInfo.attributes["peerid"]
                                val peerId = if (peerIdBytes != null) {
                                    String(peerIdBytes, Charsets.UTF_8)
                                } else {
                                    ""
                                }

                                val multiaddrBytes = serviceInfo.attributes["multiaddr"]
                                val multiaddr = if (multiaddrBytes != null) {
                                    String(multiaddrBytes, Charsets.UTF_8)
                                } else {
                                    ""
                                }

                                val eventData = mutableMapOf<String, Any>(
                                    "event" to "found",
                                    "address" to address,
                                )
                                if (peerId.isNotEmpty()) {
                                    eventData["peerId"] = peerId
                                }
                                if (multiaddr.isNotEmpty()) {
                                    eventData["multiaddr"] = multiaddr
                                }
                                sendEvent(eventData)
                            }
                        }
                    }
                    runCatching { nsdManager?.resolveService(serviceInfo, resolveListener) }
                }

                override fun onServiceLost(serviceInfo: NsdServiceInfo) {
                    sendEvent(mapOf("event" to "lost"))
                }

                override fun onDiscoveryStopped(serviceType: String) {
                    isDiscovering = false
                    sendEvent(mapOf("event" to "stopped"))
                }

                override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
                    isDiscovering = false
                    runOnUiThread {
                        runCatching { eventSink?.error("DISCOVERY_FAILED", "Error code: $errorCode", null) }
                    }
                }

                override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {}
            }

            nsdManager?.discoverServices("_isotope._tcp.", NsdManager.PROTOCOL_DNS_SD, discoveryListener)
        } catch (e: Exception) {
            isDiscovering = false
            runOnUiThread {
                runCatching { eventSink?.error("DISCOVERY_ERROR", e.message, null) }
            }
        }
    }

    private fun stopDiscovery() {
        try {
            discoveryListener?.let {
                nsdManager?.stopServiceDiscovery(it)
            }
            isDiscovering = false
        } catch (_: Exception) {}
    }

    private fun stopAnnounce(result: MethodChannel.Result) {
        try {
            registrationListener?.let {
                nsdManager?.unregisterService(it)
            }
            runOnUiThread {
                runCatching { result.success(null) }
            }
        } catch (e: Exception) {
            runOnUiThread {
                runCatching { result.error("STOP_ANNOUNCE_ERROR", e.message, null) }
            }
        }
    }

    // ==================== BLE ADVERTISING ====================

    private fun startBleAdvertising(deviceName: String, result: MethodChannel.Result) {
        try {
            if (bluetoothAdapter == null || !bluetoothAdapter!!.isEnabled) {
                result.error("BLE_DISABLED", "Bluetooth is disabled", null)
                return
            }

            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                if (checkSelfPermission(Manifest.permission.BLUETOOTH_ADVERTISE) != PackageManager.PERMISSION_GRANTED) {
                    result.error("NO_PERMISSION", "BLUETOOTH_ADVERTISE permission not granted", null)
                    return
                }
            }

            bleAdvertiser = bluetoothAdapter!!.bluetoothLeAdvertiser
            if (bleAdvertiser == null) {
                result.error("BLE_UNSUPPORTED", "BLE advertising not supported", null)
                return
            }

            stopBleAdvertisingInternal()

            val settings = AdvertiseSettings.Builder()
                .setAdvertiseMode(AdvertiseSettings.ADVERTISE_MODE_LOW_LATENCY)
                .setTxPowerLevel(AdvertiseSettings.ADVERTISE_TX_POWER_HIGH)
                .setConnectable(true)
                .setTimeout(0)
                .build()

            val data = AdvertiseData.Builder()
                .addServiceUuid(ParcelUuid(ISOTOPE_SERVICE_UUID))
                .build()

            advertiseCallback = object : AdvertiseCallback() {
                override fun onStartSuccess(settingsInEffect: AdvertiseSettings) {
                    runOnUiThread {
                        runCatching { result.success("advertising_started") }
                    }
                }

                override fun onStartFailure(errorCode: Int) {
                    runOnUiThread {
                        runCatching { result.error("ADVERTISE_FAILED", "Error code: $errorCode", null) }
                    }
                }
            }

            bleAdvertiser!!.startAdvertising(settings, data, advertiseCallback)
        } catch (e: Exception) {
            runOnUiThread {
                runCatching { result.error("BLE_ERROR", e.message, null) }
            }
        }
    }

    private fun stopBleAdvertising(result: MethodChannel.Result) {
        stopBleAdvertisingInternal()
        runOnUiThread {
            runCatching { result.success(null) }
        }
    }

    private fun stopBleAdvertisingInternal() {
        try {
            bleAdvertiser?.stopAdvertising(advertiseCallback)
            advertiseCallback = null
        } catch (_: Exception) {}
    }

    // ==================== GATT SERVER ====================

    private fun startGattServer(result: MethodChannel.Result) {
        try {
            if (bluetoothAdapter == null) {
                result.error("BLE_DISABLED", "Bluetooth is disabled", null)
                return
            }

            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                if (checkSelfPermission(Manifest.permission.BLUETOOTH_CONNECT) != PackageManager.PERMISSION_GRANTED) {
                    result.error("NO_PERMISSION", "BLUETOOTH_CONNECT permission not granted", null)
                    return
                }
            }

            val bluetoothManager = getSystemService(Context.BLUETOOTH_SERVICE) as BluetoothManager
            gattServer = bluetoothManager.openGattServer(this, object : BluetoothGattServerCallback() {
                override fun onConnectionStateChange(device: BluetoothDevice, status: Int, newState: Int) {
                    sendEvent(mapOf("event" to "ble_connection", "status" to status, "state" to newState))
                }

                override fun onCharacteristicReadRequest(
                    device: BluetoothDevice,
                    requestId: Int,
                    offset: Int,
                    characteristic: BluetoothGattCharacteristic
                ) {
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                        if (checkSelfPermission(Manifest.permission.BLUETOOTH_CONNECT) == PackageManager.PERMISSION_GRANTED) {
                            gattServer?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, offset, byteArrayOf())
                        }
                    }
                }

                override fun onCharacteristicWriteRequest(
                    device: BluetoothDevice,
                    requestId: Int,
                    characteristic: BluetoothGattCharacteristic,
                    preparedWrite: Boolean,
                    responseNeeded: Boolean,
                    offset: Int,
                    value: ByteArray
                ) {
                    val text = String(value, Charsets.UTF_8)
                    sendEvent(mapOf("event" to "ble_message", "text" to text))

                    if (responseNeeded) {
                        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                            if (checkSelfPermission(Manifest.permission.BLUETOOTH_CONNECT) == PackageManager.PERMISSION_GRANTED) {
                                gattServer?.sendResponse(device, requestId, BluetoothGatt.GATT_SUCCESS, offset, null)
                            }
                        }
                    }
                }
            })

            val service = BluetoothGattService(
                ISOTOPE_SERVICE_UUID,
                BluetoothGattService.SERVICE_TYPE_PRIMARY
            )

            val characteristic = BluetoothGattCharacteristic(
                ISOTOPE_CHAR_UUID,
                BluetoothGattCharacteristic.PROPERTY_WRITE or
                    BluetoothGattCharacteristic.PROPERTY_WRITE_NO_RESPONSE or
                    BluetoothGattCharacteristic.PROPERTY_READ or
                    BluetoothGattCharacteristic.PROPERTY_NOTIFY,
                BluetoothGattCharacteristic.PERMISSION_WRITE or
                    BluetoothGattCharacteristic.PERMISSION_READ
            )

            service.addCharacteristic(characteristic)

            val added = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                if (checkSelfPermission(Manifest.permission.BLUETOOTH_CONNECT) == PackageManager.PERMISSION_GRANTED) {
                    gattServer?.addService(service) ?: false
                } else {
                    false
                }
            } else {
                gattServer?.addService(service) ?: false
            }

            runOnUiThread {
                runCatching {
                    if (added) {
                        result.success("gatt_server_started")
                    } else {
                        result.error("GATT_ADD_FAILED", "Failed to add service", null)
                    }
                }
            }
        } catch (e: Exception) {
            runOnUiThread {
                runCatching { result.error("GATT_ERROR", e.message, null) }
            }
        }
    }
}