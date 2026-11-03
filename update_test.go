package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func postUpdate(pass string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/admin/update", strings.NewReader(url.Values{"password": {pass}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	serveUpdate(w, r)
	return w
}

func TestUpdatePassword(t *testing.T) {
	fails = nil
	t.Setenv("ADMIN_PASSWORD", "")
	if c := postUpdate("x").Code; c != http.StatusForbidden {
		t.Fatalf("sans ADMIN_PASSWORD la mise à jour doit être désactivée, obtenu %d", c)
	}
	t.Setenv("ADMIN_PASSWORD", "secret")
	get := httptest.NewRecorder()
	serveUpdate(get, httptest.NewRequest("GET", "/admin/update", nil))
	if get.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET refusé attendu, obtenu %d", get.Code)
	}
	for range maxFails {
		if c := postUpdate("mauvais").Code; c != http.StatusForbidden {
			t.Fatalf("mauvais mot de passe : 403 attendu, obtenu %d", c)
		}
	}
	// même avec le bon mot de passe : bloqué après trop d'essais ratés
	if c := postUpdate("secret").Code; c != http.StatusTooManyRequests {
		t.Fatalf("blocage attendu, obtenu %d", c)
	}
	fails = nil
}

func TestSelfUpdate(t *testing.T) {
	bin := []byte("nouveau binaire")
	sum := sha256hex(bin)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/SHA256SUMS":
			w.Write([]byte(sum + "  " + releaseAsset + "\n"))
		case "/" + releaseAsset:
			w.Write(bin)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := releaseURL
	releaseURL = srv.URL + "/"
	defer func() { releaseURL = old }()

	exe := filepath.Join(t.TempDir(), "patate")
	os.WriteFile(exe, []byte("ancien"), 0o755)
	if changed, err := selfUpdate(exe); err != nil || !changed {
		t.Fatalf("mise à jour attendue, obtenu %v / %v", changed, err)
	}
	if got, _ := os.ReadFile(exe); string(got) != string(bin) {
		t.Fatalf("binaire non remplacé : %q", got)
	}
	if changed, err := selfUpdate(exe); err != nil || changed {
		t.Fatalf("déjà à jour attendu, obtenu %v / %v", changed, err)
	}

	sum = strings.Repeat("0", 64) // somme qui ne correspond pas : rien ne doit être écrit
	os.WriteFile(exe, []byte("ancien"), 0o755)
	if _, err := selfUpdate(exe); err == nil || !strings.Contains(err.Error(), "invalide") {
		t.Fatalf("somme invalide attendue, obtenu %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "ancien" {
		t.Fatal("le binaire ne doit pas changer si la somme est fausse")
	}
}

func TestSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	secure(http.FileServerFS(files)).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy"} {
		if w.Header().Get(h) == "" {
			t.Errorf("en-tête %s manquant", h)
		}
	}
}
