// Mise à jour depuis le site : /admin.html envoie le mot de passe ADMIN_PASSWORD à /admin/update.
// Le serveur télécharge la dernière release GitHub, vérifie sa somme SHA-256, remplace son propre
// binaire puis se relance sur place (syscall.Exec : même PID, le service ne voit rien).
package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	releaseAsset = "patate-linux-amd64"
	maxFails     = 5 // mauvais mots de passe tolérés par fenêtre de 15 min, tous visiteurs confondus
	failWindow   = 15 * time.Minute
)

var releaseURL = "https://github.com/LBouzac/patate/releases/download/latest/" // variable pour les tests

var (
	adminMu  sync.Mutex
	fails    []time.Time
	updating bool
)

// version : commit et date de compilation (renseignés par go build dans un dépôt Git).
func version() string {
	rev, at := "dev", ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value[:min(7, len(s.Value))]
			case "vcs.time":
				at = " du " + s.Value
			}
		}
	}
	return rev + at
}

func serveVersion(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, version()) }

func serveUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST uniquement", http.StatusMethodNotAllowed)
		return
	}
	pass := os.Getenv("ADMIN_PASSWORD")
	if pass == "" {
		http.Error(w, "Mise à jour désactivée : ADMIN_PASSWORD n'est pas défini sur le serveur", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	given := r.PostFormValue("password")

	adminMu.Lock()
	fails = slices.DeleteFunc(fails, func(t time.Time) bool { return time.Since(t) > failWindow })
	if len(fails) >= maxFails {
		adminMu.Unlock()
		http.Error(w, "Trop d'essais ratés, réessaie dans 15 minutes", http.StatusTooManyRequests)
		return
	}
	a, b := sha256.Sum256([]byte(given)), sha256.Sum256([]byte(pass)) // même longueur : comparaison en temps constant
	if subtle.ConstantTimeCompare(a[:], b[:]) != 1 {
		fails = append(fails, time.Now())
		adminMu.Unlock()
		http.Error(w, "Mot de passe incorrect", http.StatusForbidden)
		return
	}
	if updating {
		adminMu.Unlock()
		http.Error(w, "Mise à jour déjà en cours", http.StatusConflict)
		return
	}
	updating = true
	adminMu.Unlock()

	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	changed := false
	if err == nil {
		changed, err = selfUpdate(exe)
	}
	if err != nil || !changed {
		adminMu.Lock()
		updating = false
		adminMu.Unlock()
		if err != nil {
			log.Println("mise à jour :", err)
			http.Error(w, "Échec : "+err.Error(), http.StatusBadGateway)
		} else {
			fmt.Fprint(w, "Déjà à jour (", version(), ")")
		}
		return
	}
	log.Println("mise à jour installée, redémarrage")
	fmt.Fprint(w, "Mise à jour installée, redémarrage…")
	go func() {
		time.Sleep(time.Second) // laisse partir la réponse
		log.Fatal(syscall.Exec(exe, os.Args, os.Environ()))
	}()
}

// selfUpdate télécharge la release, vérifie la somme et remplace exe. false si c'est déjà la même version.
func selfUpdate(exe string) (bool, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return false, errors.New("seul Linux amd64 est publié")
	}
	sums, err := fetch(releaseURL+"SHA256SUMS", 4096)
	if err != nil {
		return false, err
	}
	want := ""
	for _, line := range strings.Split(string(sums), "\n") {
		if f := strings.Fields(line); len(f) == 2 && strings.TrimPrefix(f[1], "*") == releaseAsset {
			want = f[0]
		}
	}
	if want == "" {
		return false, errors.New("somme de contrôle absente de SHA256SUMS")
	}
	if cur, err := os.ReadFile(exe); err == nil && sha256hex(cur) == want {
		return false, nil
	}
	bin, err := fetch(releaseURL+releaseAsset, 64<<20)
	if err != nil {
		return false, err
	}
	if sha256hex(bin) != want {
		return false, errors.New("somme de contrôle invalide : téléchargement corrompu")
	}
	// écrire à côté puis renommer : on ne peut pas écrire sur un binaire en cours d'exécution
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, bin, 0o755); err != nil {
		return false, fmt.Errorf("écriture impossible (le dossier du binaire doit appartenir au service) : %w", err)
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Remove(tmp)
		return false, err
	}
	return true, nil
}

func sha256hex(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func fetch(url string, max int64) ([]byte, error) {
	c := http.Client{Timeout: 3 * time.Minute}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s : %s", url, resp.Status)
	}
	var buf bytes.Buffer
	if n, err := io.Copy(&buf, io.LimitReader(resp.Body, max+1)); err != nil {
		return nil, err
	} else if n > max {
		return nil, fmt.Errorf("%s : fichier trop gros", url)
	}
	return buf.Bytes(), nil
}
