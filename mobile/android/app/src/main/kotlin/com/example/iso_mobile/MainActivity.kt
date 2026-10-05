// mobile/android/app/src/main/kotlin/com/example/iso_mobile/MainActivity.kt
package com.example.iso_mobile

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
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
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.os.ParcelUuid
import android.util.Log
import androidx.core.app.ActivityCompat
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
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

    // Уведомления и foreground-состояние.
    private var isForeground: Boolean = true
    private var pendingOpenChat: String? = null
    private var notificationChannelCreated: Boolean = false
    private val NOTIFICATION_CHANNEL_ID = "isotope_messages"
    private val NOTIFICATION_CHANNEL_NAME = "Сообщения"
    private val NOTIFICATION_REQUEST_CODE = 2001
    private val NOTIFICATION_PERMISSION_REQUEST_CODE = 2002

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        nsdManager = getSystemService(Context.NSD_SERVICE) as NsdManager

        Mobile.setFilesDir(filesDir.absolutePath)
        IsotopeService.start(this)
        createNotificationChannel()
        requestNotificationPermissionIfNeeded()

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
                        val period = call.argument<String>("period") ?: "forever"
                        val mode = call.argument<String>("mode") ?: ""
                        Thread {
                            val response = Mobile.sendMessage(text, period, mode)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "sendToPeer" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        val text = call.argument<String>("text") ?: ""
                        val period = call.argument<String>("period") ?: "forever"
                        val mode = call.argument<String>("mode") ?: ""
                        Thread {
                            val response = Mobile.sendToPeer(peerID, text, period, mode)
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
                    "connectToPeerWithFallback" -> {
                        val multiaddrsJSON = call.argument<String>("multiaddrsJSON") ?: ""
                        Thread {
                            val response = Mobile.connectToPeerWithFallback(multiaddrsJSON)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "announce" -> {
                        val multiaddrsJSON = call.argument<String>("multiaddrsJSON") ?: ""
                        Thread {
                            val response = Mobile.announce(multiaddrsJSON)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "findPeerByID" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        Thread {
                            val response = Mobile.findPeerByID(peerID)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getMyQRData" -> {
                        Thread {
                            val response = Mobile.getMyQRData()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getEd25519PublicKey" -> {
                        Thread {
                            val response = Mobile.getEd25519PublicKey()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getX25519PublicKey" -> {
                        Thread {
                            val response = Mobile.getX25519PublicKey()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
"addContact" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        val ed25519Pub = call.argument<String>("ed25519Pub") ?: ""
                        val x25519Pub = call.argument<String>("x25519Pub") ?: ""
                        val signature = call.argument<String>("signature") ?: ""
                        val localName = call.argument<String>("localName") ?: ""
                        val remoteName = call.argument<String>("remoteName") ?: ""
                        val readEnabled = call.argument<Boolean>("readEnabled") ?: true
                        Thread {
                            val response = Mobile.addContact(peerID, ed25519Pub, x25519Pub, signature, localName, remoteName, readEnabled)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "sendContactHello" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        Thread {
                            val response = Mobile.sendContactHello(peerID)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "sendContactRequest" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        val name = call.argument<String>("name") ?: ""
                        Thread {
                            val response = Mobile.sendContactRequest(peerID, name)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getRequests" -> {
                        Thread {
                            val response = Mobile.getRequests()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "acceptRequestByID" -> {
                        val id = call.argument<String>("id") ?: ""
                        Thread {
                            val response = Mobile.acceptRequestByID(id)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "rejectRequestByID" -> {
                        val id = call.argument<String>("id") ?: ""
                        Thread {
                            val response = Mobile.rejectRequestByID(id)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "sendRead" -> {
                        val ref = call.argument<String>("ref") ?: ""
                        val recipient = call.argument<String>("recipient") ?: ""
                        Thread {
                            val response = Mobile.sendRead(ref, recipient)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "sendReadBatch" -> {
                        val refsJson = call.argument<String>("refs") ?: "[]"
                        val recipient = call.argument<String>("recipient") ?: ""
                        Thread {
                            val response = Mobile.sendReadBatch(refsJson, recipient)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "markReadLocally" -> {
                        val refsJson = call.argument<String>("refs") ?: "[]"
                        Thread {
                            val response = Mobile.markReadLocally(refsJson)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getContacts" -> {
                        Thread {
                            val response = Mobile.getContacts()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getMessageStatuses" -> {
                        Thread {
                            val response = Mobile.getMessageStatuses()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getContact" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        Thread {
                            val response = Mobile.getContact(peerID)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "setMyReadEnabled" -> {
                        val enabled = call.argument<Boolean>("enabled") ?: true
                        Thread {
                            val response = Mobile.setMyReadEnabled(enabled)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getMyReadEnabled" -> {
                        Thread {
                            val response = Mobile.getMyReadEnabled()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "setMyDisplayName" -> {
                        val name = call.argument<String>("name") ?: ""
                        Thread {
                            val response = Mobile.setMyDisplayName(name)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getMyDisplayName" -> {
                        Thread {
                            val response = Mobile.getMyDisplayName()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getShowNotificationContent" -> {
                        Thread {
                            val response = Mobile.getShowNotificationContent()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "setShowNotificationContent" -> {
                        val enabled = call.argument<Boolean>("enabled") ?: true
                        Thread {
                            val response = Mobile.setShowNotificationContent(enabled)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "setTtl" -> {
                        val period = call.argument<String>("period") ?: "forever"
                        val mode = call.argument<String>("mode") ?: ""
                        Thread {
                            val response = Mobile.setTtl(period, mode)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getTtl" -> {
                        Thread {
                            val response = Mobile.getTtl()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "renameContact" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        val localName = call.argument<String>("localName") ?: ""
                        Thread {
                            val response = Mobile.renameContact(peerID, localName)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "removeContact" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        Thread {
                            val response = Mobile.removeContact(peerID)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "getDeletedPeers" -> {
                        Thread {
                            val response = Mobile.getDeletedPeers()
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "removeFromDeleted" -> {
                        val peerID = call.argument<String>("peerID") ?: ""
                        Thread {
                            val response = Mobile.removeFromDeleted(peerID)
                            runOnUiThread { result.success(response) }
                        }.start()
                    }
                    "setSecureFlag" -> {
                        val enabled = call.argument<Boolean>("enabled") ?: false
                        runOnUiThread {
                            if (enabled) {
                                window.setFlags(
                                    android.view.WindowManager.LayoutParams.FLAG_SECURE,
                                    android.view.WindowManager.LayoutParams.FLAG_SECURE
                                )
                            } else {
                                window.clearFlags(
                                    android.view.WindowManager.LayoutParams.FLAG_SECURE
                                )
                            }
                            result.success("ok")
                        }
                    }
                    "getFilesDir" -> {
                        result.success(filesDir.absolutePath)
                    }
                    "getPendingOpenChat" -> {
                        val peer = pendingOpenChat ?: ""
                        pendingOpenChat = null
                        result.success(peer)
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
                    "isIgnoringBatteryOptimizations" -> {
                        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
                            val pm = getSystemService(Context.POWER_SERVICE) as android.os.PowerManager
                            result.success(pm.isIgnoringBatteryOptimizations(packageName))
                        } else {
                            result.success(true)
                        }
                    }
                    "requestIgnoreBatteryOptimizations" -> {
                        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
                            val pm = getSystemService(Context.POWER_SERVICE) as android.os.PowerManager
                            if (pm.isIgnoringBatteryOptimizations(packageName)) {
                                result.success("already_ignoring")
                            } else {
                                try {
                                    val intent = android.content.Intent(
                                        android.provider.Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
                                        android.net.Uri.parse("package:$packageName")
                                    )
                                    startActivity(intent)
                                    result.success("requested")
                                } catch (e: Exception) {
                                    result.error("REQUEST_FAILED", e.message, null)
                                }
                            }
                        } else {
                            result.success("not_supported")
                        }
                    }
                    else -> result.notImplemented()
                }
            }

        // EventChannel для новых сообщений
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, MESSAGES_EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
                    messageEventSink = events
                    Mobile.setMessageCallback(object : mobile.MessageCallback {
                        override fun onMessage(message: String) {
                            runOnUiThread {
                                runCatching { messageEventSink?.success(message) }
                                if (!isForeground) {
                                    showMessageNotification(message)
                                }
                            }
                        }
                    })
                }

                override fun onCancel(arguments: Any?) {
                    Mobile.setMessageCallback(null)
                    messageEventSink = null
                }
            })

        val bluetoothManager = getSystemService(Context.BLUETOOTH_SERVICE) as BluetoothManager
        bluetoothAdapter = bluetoothManager.adapter
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        handleIntentForChatOpen(intent)
    }

    override fun onResume() {
        super.onResume()
        isForeground = true
    }

    override fun onPause() {
        super.onPause()
        isForeground = false
    }

    override fun onDestroy() {
        Log.d("MainActivity", "onDestroy() — останавливаем IsotopeService")
        IsotopeService.stop(this)
        super.onDestroy()
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

    private fun createNotificationChannel() {
        if (notificationChannelCreated) return
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                NOTIFICATION_CHANNEL_ID,
                NOTIFICATION_CHANNEL_NAME,
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Уведомления о новых сообщениях"
            }
            val nm = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            nm.createNotificationChannel(channel)
        }
        notificationChannelCreated = true
    }

    private fun requestNotificationPermissionIfNeeded() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ActivityCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS)
                != PackageManager.PERMISSION_GRANTED) {
                ActivityCompat.requestPermissions(
                    this,
                    arrayOf(Manifest.permission.POST_NOTIFICATIONS),
                    NOTIFICATION_PERMISSION_REQUEST_CODE
                )
            }
        }
    }

    private fun showMessageNotification(messageJson: String) {
        try {
            val map = org.json.JSONObject(messageJson)

            // Только входящие, не свои, не служебные.
            val isOwn = map.optBoolean("isOwn", false)
            val type = map.optInt("type", 0)
            if (isOwn || type != 0) return

            val sender = map.optString("sender", "")
            if (sender.isEmpty()) return

            val senderName = map.optString("sender_name", "")
            val displaySender = if (senderName.isNotEmpty()) senderName
                                else if (sender.length > 12) sender.substring(0, 12) + "…"
                                else sender

            // Заголовок и текст.
            val ttlMode = map.optString("ttl_mode", "")
            val showContentResp = Mobile.getShowNotificationContent()
            val showContent = showContentResp.contains("\"show_notification_content\":true")

            val title = displaySender
            val body = if (!showContent) {
                "Новое сообщение"
            } else if (ttlMode.isNotEmpty()) {
                "Исчезающее сообщение"
            } else {
                val text = if (map.optString("plainText", "").isNotEmpty() && isOwn) {
                    map.optString("plainText")
                } else {
                    map.optString("text", "")
                }
                if (text.length > 50) text.substring(0, 50) + "…" else text
            }

            // Тап → открыть чат.
            val intent = Intent(this, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
                putExtra("peer_id", sender)
            }
            val pendingIntent = PendingIntent.getActivity(
                this,
                sender.hashCode(),
                intent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )

            val builder = NotificationCompat.Builder(this, NOTIFICATION_CHANNEL_ID)
                .setSmallIcon(android.R.drawable.ic_dialog_email)
                .setContentTitle(title)
                .setContentText(body)
                .setPriority(NotificationCompat.PRIORITY_HIGH)
                .setAutoCancel(true)
                .setContentIntent(pendingIntent)

            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                if (ActivityCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS)
                    != PackageManager.PERMISSION_GRANTED) {
                    return
                }
            }
            NotificationManagerCompat.from(this).notify(sender.hashCode(), builder.build())
        } catch (e: Exception) {
            Log.e("MainActivity", "showMessageNotification error: ${e.message}")
        }
    }

    private fun handleIntentForChatOpen(intent: Intent?) {
        val peerId = intent?.getStringExtra("peer_id") ?: return
        if (peerId.isEmpty()) return
        pendingOpenChat = peerId
        runCatching { messageEventSink?.success("{\"open_chat\":\"$peerId\"}") }
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
// mobile/android/app/src/main/kotlin/com/example/iso_mobile/MainActivity.kt