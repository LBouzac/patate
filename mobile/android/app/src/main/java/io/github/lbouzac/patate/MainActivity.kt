package io.github.lbouzac.patate

import android.net.Uri
import android.os.Bundle
import android.view.WindowManager
import android.webkit.JavascriptInterface
import android.webkit.ValueCallback
import android.webkit.WebChromeClient
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.ComponentActivity
import androidx.activity.addCallback
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import com.google.mlkit.vision.codescanner.GmsBarcodeScanning
import org.json.JSONObject

// Une WebView : la page d'accueil (assets/start.html) puis le serveur Patate choisi.
class MainActivity : ComponentActivity() {
    private lateinit var web: WebView
    private var fileCallback: ValueCallback<Array<Uri>>? = null
    private val pickFile = registerForActivityResult(ActivityResultContracts.StartActivityForResult()) {
        fileCallback?.onReceiveValue(WebChromeClient.FileChooserParams.parseResult(it.resultCode, it.data))
        fileCallback = null
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // en http, le navigateur refuse le wakeLock des pages : l'app garde l'écran allumé à leur place
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        web = WebView(this)
        web.setBackgroundColor(0xFF020617.toInt())
        setContentView(web)
        ViewCompat.setOnApplyWindowInsetsListener(web) { v, insets ->
            val b = insets.getInsets(WindowInsetsCompat.Type.systemBars() or WindowInsetsCompat.Type.displayCutout())
            v.setPadding(b.left, b.top, b.right, b.bottom)
            insets
        }
        web.settings.javaScriptEnabled = true
        web.settings.domStorageEnabled = true
        web.settings.mediaPlaybackRequiresUserGesture = false
        web.addJavascriptInterface(Bridge(), "PatateApp")
        web.webViewClient = WebViewClient() // les liens restent dans l'app
        web.webChromeClient = object : WebChromeClient() { // alert/confirm/prompt, et l'envoi de photo (Dessine)
            override fun onShowFileChooser(view: WebView, callback: ValueCallback<Array<Uri>>, params: FileChooserParams): Boolean {
                fileCallback?.onReceiveValue(null)
                fileCallback = callback
                pickFile.launch(params.createIntent())
                return true
            }
        }
        onBackPressedDispatcher.addCallback(this) { if (web.canGoBack()) web.goBack() else finish() }
        web.loadUrl("file:///android_asset/start.html")
    }

    inner class Bridge {
        @JavascriptInterface
        fun scan() = runOnUiThread {
            GmsBarcodeScanning.getClient(this@MainActivity).startScan()
                .addOnSuccessListener { reply(it.rawValue ?: "", "") }
                .addOnFailureListener { reply("", it.message ?: "erreur") }
        }
    }

    private fun reply(text: String, error: String) =
        web.evaluateJavascript("onScan(${JSONObject.quote(text)}, ${JSONObject.quote(error)})", null)
}
