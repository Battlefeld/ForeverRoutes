package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BundleManifest describes a bundle: its version, the zip's SHA-256 and every file in it with its SHA-256.
// ForeverCore downloads the bundle of the latest release, checks both and only then replaces its ForeverRoutes folder.
type BundleManifest struct {
	Schema  string            `json:"schema"`
	Version string            `json:"version"`
	ZipSHA  string            `json:"zipSHA256"`
	Files   map[string]string `json:"files"`
}

// cmdBundle packs what ForeverCore needs into dist/foreverroutes-bundle.zip and its manifest
// dist/foreverroutes-bundle.json: the dataset, the walk grid and table, the routes, the kill profiles and, when given, the
// Windows planner.
func cmdBundle(args []string) error {
	fs := newFlags("bundle")
	out := fs.String("out", "dist", "folder to write the bundle and its manifest to")
	version := fs.String("version", "", "version (the release tag); default: the time")
	exe := fs.String("exe", "", "foreverroutes.exe to include (built for Windows)")
	fs.Parse(args)
	if *version == "" {
		*version = time.Now().UTC().Format("20060102-150405")
	}
	files := map[string]string{ // name in the bundle -> source
		"routes-data.json.gz": "data/routes-data.json.gz",
		"walks.json.gz":       "data/walks.json.gz",
		"walktable.json.gz":   "data/walktable.json.gz",
		"README-data.md":      "data/README.md",
	}
	for _, pat := range []string{"data/routes/*.json", "data/routes/README.md", "data/profiles/*.json"} {
		m, err := filepath.Glob(pat)
		if err != nil {
			return err
		}
		for _, p := range m {
			rel, _ := filepath.Rel("data", p)
			files[filepath.ToSlash(rel)] = p
		}
	}
	if *exe != "" {
		files["foreverroutes.exe"] = *exe
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	zpath := filepath.Join(*out, "foreverroutes-bundle.zip")
	zf, err := os.Create(zpath)
	if err != nil {
		return err
	}
	man := BundleManifest{Schema: "foreverroutes.bundle.v1", Version: *version, Files: map[string]string{}}
	zw := zip.NewWriter(zf)
	for _, n := range names {
		b, err := os.ReadFile(files[n])
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		man.Files[n] = hex.EncodeToString(sum[:])
		w, err := zw.CreateHeader(&zip.FileHeader{Name: n, Method: zip.Deflate, Modified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := zf.Close(); err != nil {
		return err
	}
	zb, err := os.ReadFile(zpath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(zb)
	man.ZipSHA = hex.EncodeToString(sum[:])
	mb, _ := json.MarshalIndent(man, "", " ")
	if err := os.WriteFile(filepath.Join(*out, "foreverroutes-bundle.json"), append(mb, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "bundle %s: %d files, %d bytes\n", *version, len(names), len(zb))
	return nil
}
