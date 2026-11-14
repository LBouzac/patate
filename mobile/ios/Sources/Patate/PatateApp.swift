import SwiftUI
import VisionKit
import WebKit

@main
struct PatateApp: App {
    var body: some Scene {
        WindowGroup {
            Web().background(Color(red: 2 / 255, green: 6 / 255, blue: 23 / 255)).preferredColorScheme(.dark)
        }
    }
}

struct Web: UIViewControllerRepresentable {
    func makeUIViewController(context: Context) -> WebController { WebController() }
    func updateUIViewController(_ controller: WebController, context: Context) {}
}

// Une WebView : la page d'accueil (start.html) puis le serveur Patate choisi.
final class WebController: UIViewController, WKScriptMessageHandler, WKUIDelegate, DataScannerViewControllerDelegate {
    private var web: WKWebView!

    override func loadView() {
        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true
        config.mediaTypesRequiringUserActionForPlayback = []
        config.userContentController.add(self, name: "scan")
        web = WKWebView(frame: .zero, configuration: config)
        web.uiDelegate = self
        web.allowsBackForwardNavigationGestures = true // glisser depuis le bord : retour, jusqu'à l'accueil
        web.isOpaque = false
        view = web
        // en http, Safari refuse le wakeLock des pages : l'app garde l'écran allumé à leur place
        UIApplication.shared.isIdleTimerDisabled = true
        let start = Bundle.main.url(forResource: "start", withExtension: "html")!
        web.loadFileURL(start, allowingReadAccessTo: start)
    }

    // --- scan du QR code (bouton de start.html), réponse par onScan(texte, erreur)

    func userContentController(_ controller: WKUserContentController, didReceive message: WKScriptMessage) {
        guard DataScannerViewController.isSupported else { return reply("", "pas de scanner sur cet appareil") }
        let scanner = DataScannerViewController(recognizedDataTypes: [.barcode(symbologies: [.qr])], isHighlightingEnabled: true)
        scanner.delegate = self
        present(scanner, animated: true) {
            do { try scanner.startScanning() } catch {
                scanner.dismiss(animated: true)
                self.reply("", "caméra refusée (Réglages > Patate)")
            }
        }
    }

    func dataScanner(_ scanner: DataScannerViewController, didAdd added: [RecognizedItem], allItems: [RecognizedItem]) {
        guard case .barcode(let code)? = added.first, let text = code.payloadStringValue else { return }
        scanner.stopScanning()
        scanner.dismiss(animated: true)
        reply(text, "")
    }

    private func reply(_ text: String, _ error: String) {
        let args = try! JSONSerialization.data(withJSONObject: [text, error])
        web.evaluateJavaScript("onScan(...\(String(decoding: args, as: UTF8.self)))")
    }

    // --- alert / confirm / prompt : WKWebView ne les affiche pas tout seul

    private func dialog(_ message: String, field: String? = nil, cancel: Bool, done: @escaping (Bool, String?) -> Void) {
        let alert = UIAlertController(title: nil, message: message, preferredStyle: .alert)
        if let field { alert.addTextField { $0.text = field } }
        if cancel { alert.addAction(UIAlertAction(title: "Annuler", style: .cancel) { _ in done(false, nil) }) }
        alert.addAction(UIAlertAction(title: "OK", style: .default) { _ in done(true, alert.textFields?.first?.text) })
        present(alert, animated: true)
    }

    func webView(_ webView: WKWebView, runJavaScriptAlertPanelWithMessage message: String,
                 initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping @MainActor () -> Void) {
        dialog(message, cancel: false) { _, _ in completionHandler() }
    }

    func webView(_ webView: WKWebView, runJavaScriptConfirmPanelWithMessage message: String,
                 initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping @MainActor (Bool) -> Void) {
        dialog(message, cancel: true) { ok, _ in completionHandler(ok) }
    }

    func webView(_ webView: WKWebView, runJavaScriptTextInputPanelWithPrompt prompt: String, defaultText: String?,
                 initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping @MainActor (String?) -> Void) {
        dialog(prompt, field: defaultText ?? "", cancel: true) { ok, text in completionHandler(ok ? text : nil) }
    }
}
