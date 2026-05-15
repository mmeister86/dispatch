package clipboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	ErrNoImages    = errors.New("keine Bilder in der Zwischenablage gefunden")
	ErrUnavailable = errors.New("Clipboard-Bildzugriff ist nicht verfuegbar")
)

type Attachment struct {
	Path         string
	OriginalName string
	MIMEType     string
	Source       string
}

type Runner func(context.Context, string, ...string) ([]byte, error)
type NativeDarwinPNGReader func(context.Context) ([]byte, error)

type Reader struct {
	GOOS                string
	Runner              Runner
	ReadNativeDarwinPNG NativeDarwinPNGReader
	CacheDir            string
	Now                 func() time.Time
}

func NewReader() Reader {
	return Reader{GOOS: runtime.GOOS, Runner: runCommand}
}

func (r Reader) ReadImages(ctx context.Context) ([]Attachment, error) {
	if err := r.prepare(); err != nil {
		return nil, err
	}

	switch r.GOOS {
	case "darwin":
		return r.readDarwin(ctx)
	case "linux":
		return r.readLinux(ctx)
	case "windows":
		return r.readWindows(ctx)
	default:
		return nil, fmt.Errorf("%w: Betriebssystem %s wird nicht unterstuetzt", ErrUnavailable, r.GOOS)
	}
}

func (r Reader) AttachPastedImagePaths(ctx context.Context, value string) ([]Attachment, error) {
	if err := r.prepare(); err != nil {
		return nil, err
	}
	attachments := r.attachFiles(ctx, parsePastedPaths(value))
	if len(attachments) == 0 {
		return nil, ErrNoImages
	}
	return attachments, nil
}

func (r *Reader) prepare() error {
	if r.GOOS == "" {
		r.GOOS = runtime.GOOS
	}
	if r.Runner == nil {
		r.Runner = runCommand
	}
	if r.ReadNativeDarwinPNG == nil {
		r.ReadNativeDarwinPNG = readNativeDarwinPNG
	}
	if r.Now == nil {
		r.Now = func() time.Time { return time.Now().UTC() }
	}
	cacheDir, err := r.cacheDir()
	if err != nil {
		return err
	}
	r.CacheDir = cacheDir
	return nil
}

func (r Reader) readDarwin(ctx context.Context) ([]Attachment, error) {
	var attachments []Attachment
	attachments = append(attachments, r.attachFiles(ctx, r.darwinFilePaths(ctx))...)
	image, err := r.Runner(ctx, "pngpaste", "-")
	if err == nil && isPNG(image) {
		if attachment, err := r.saveImageBytes(image, "clipboard.png", "image/png", "clipboard"); err == nil {
			attachments = append(attachments, attachment)
		}
	}
	if len(attachments) == 0 && errors.Is(err, ErrUnavailable) {
		if image, nativeErr := r.ReadNativeDarwinPNG(ctx); nativeErr == nil && isPNG(image) {
			if attachment, err := r.saveImageBytes(image, "clipboard.png", "image/png", "clipboard"); err == nil {
				attachments = append(attachments, attachment)
			}
		}
	}
	if len(attachments) == 0 {
		if errors.Is(err, ErrUnavailable) && r.darwinClipboardHasBitmap(ctx) {
			return nil, fmt.Errorf("%w: macOS-Bilddaten konnten nicht aus der Zwischenablage gelesen werden", ErrUnavailable)
		}
		return nil, ErrNoImages
	}
	return attachments, nil
}

func (r Reader) readLinux(ctx context.Context) ([]Attachment, error) {
	var attachments []Attachment
	var unavailable int
	for _, cmd := range [][]string{
		{"wl-paste", "--type", "text/uri-list"},
		{"xclip", "-selection", "clipboard", "-t", "text/uri-list", "-o"},
		{"xsel", "--clipboard", "--output"},
	} {
		out, err := r.Runner(ctx, cmd[0], cmd[1:]...)
		if errors.Is(err, ErrUnavailable) {
			unavailable++
			continue
		}
		if err == nil {
			attachments = append(attachments, r.attachFiles(ctx, parseFileList(string(out)))...)
			if len(attachments) > 0 {
				break
			}
		}
	}
	for _, cmd := range [][]string{
		{"wl-paste", "--type", "image/png"},
		{"xclip", "-selection", "clipboard", "-t", "image/png", "-o"},
	} {
		out, err := r.Runner(ctx, cmd[0], cmd[1:]...)
		if errors.Is(err, ErrUnavailable) {
			unavailable++
			continue
		}
		if err == nil && isPNG(out) {
			if attachment, err := r.saveImageBytes(out, "clipboard.png", "image/png", "clipboard"); err == nil {
				attachments = append(attachments, attachment)
			}
			break
		}
	}
	if len(attachments) > 0 {
		return attachments, nil
	}
	if unavailable > 0 {
		return nil, fmt.Errorf("%w: installiere wl-paste, xclip oder xsel fuer Bild-Paste", ErrUnavailable)
	}
	return nil, ErrNoImages
}

func (r Reader) readWindows(ctx context.Context) ([]Attachment, error) {
	bitmapPath := filepath.Join(r.CacheDir, r.cacheName("clipboard.png", 0))
	out, err := r.Runner(ctx, "powershell", "-NoProfile", "-Sta", "-Command", windowsClipboardScript(bitmapPath))
	if err != nil {
		if errors.Is(err, ErrUnavailable) {
			return nil, fmt.Errorf("%w: PowerShell Clipboard-Zugriff ist nicht verfuegbar", ErrUnavailable)
		}
		return nil, err
	}
	var attachments []Attachment
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		kind, value, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		switch kind {
		case "file":
			attachments = append(attachments, r.attachFiles(ctx, []string{value})...)
		case "bitmap":
			if mimeType, ok := imageMIMEForPath(value); ok {
				attachments = append(attachments, Attachment{Path: value, OriginalName: filepath.Base(value), MIMEType: mimeType, Source: "clipboard"})
			}
		}
	}
	if len(attachments) == 0 {
		return nil, ErrNoImages
	}
	return attachments, nil
}

func (r Reader) darwinFilePaths(ctx context.Context) []string {
	args := []string{
		"-e", `set output to ""`,
		"-e", `try`,
		"-e", `set theFiles to the clipboard as «class furl»`,
		"-e", `repeat with f in theFiles`,
		"-e", `set output to output & POSIX path of f & linefeed`,
		"-e", `end repeat`,
		"-e", `output`,
		"-e", `on error`,
		"-e", `""`,
		"-e", `end try`,
	}
	out, err := r.Runner(ctx, "osascript", args...)
	if err != nil {
		return nil
	}
	return parseFileList(string(out))
}

func (r Reader) darwinClipboardHasBitmap(ctx context.Context) bool {
	for _, script := range []string{
		`set pngData to the clipboard as «class PNGf»`,
		`set tiffData to the clipboard as «class TIFF»`,
	} {
		if _, err := r.Runner(ctx, "osascript", "-e", script); err == nil {
			return true
		}
	}
	return false
}

func (r Reader) attachFiles(_ context.Context, paths []string) []Attachment {
	var attachments []Attachment
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		mimeType, ok := imageMIMEForPath(path)
		if !ok {
			continue
		}
		copied, err := r.copyFileToCache(path, len(attachments))
		if err != nil {
			continue
		}
		attachments = append(attachments, Attachment{
			Path:         copied,
			OriginalName: filepath.Base(path),
			MIMEType:     mimeType,
			Source:       "file",
		})
	}
	return attachments
}

func (r Reader) copyFileToCache(path string, index int) (string, error) {
	source, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()
	target := filepath.Join(r.CacheDir, r.cacheName(filepath.Base(path), index))
	dest, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer dest.Close()
	if _, err := io.Copy(dest, source); err != nil {
		return "", err
	}
	return target, nil
}

func (r Reader) saveImageBytes(data []byte, name, mimeType, source string) (Attachment, error) {
	target := filepath.Join(r.CacheDir, r.cacheName(name, 0))
	if err := os.WriteFile(target, data, 0o600); err != nil {
		return Attachment{}, err
	}
	return Attachment{Path: target, OriginalName: name, MIMEType: mimeType, Source: source}, nil
}

func (r Reader) cacheName(name string, index int) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		ext = ".png"
	}
	stem := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	stem = sanitizeName(stem)
	if stem == "" {
		stem = "image"
	}
	return fmt.Sprintf("%s-%02d-%s-%d%s", r.Now().UTC().Format("20060102T150405"), index+1, stem, time.Now().UnixNano(), ext)
}

func (r Reader) cacheDir() (string, error) {
	if strings.TrimSpace(r.CacheDir) != "" {
		if err := os.MkdirAll(r.CacheDir, 0o700); err != nil {
			return "", err
		}
		return r.CacheDir, nil
	}
	base, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "dispatch", "attachments")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func parseFileList(value string) []string {
	var paths []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "file://") {
			u, err := url.Parse(line)
			if err == nil {
				line = u.Path
			}
		}
		paths = append(paths, line)
	}
	return paths
}

func parsePastedPaths(value string) []string {
	candidates := parseFileList(value)
	if len(candidates) == 1 && candidates[0] == strings.TrimSpace(value) {
		candidates = splitShellishFields(value)
	}
	seen := make(map[string]bool, len(candidates))
	var paths []string
	for _, candidate := range candidates {
		candidate = normalizePastedPath(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		paths = append(paths, candidate)
	}
	return paths
}

func splitShellishFields(value string) []string {
	var fields []string
	var b strings.Builder
	var quote rune
	escaped := false
	for _, ch := range strings.TrimSpace(value) {
		switch {
		case escaped:
			b.WriteRune(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case quote != 0:
			if ch == quote {
				quote = 0
			} else {
				b.WriteRune(ch)
			}
		case ch == '\'' || ch == '"':
			quote = ch
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			if b.Len() > 0 {
				fields = append(fields, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(ch)
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	if b.Len() > 0 {
		fields = append(fields, b.String())
	}
	return fields
}

func normalizePastedPath(value string) string {
	value = strings.TrimSpace(strings.Trim(value, `"'`))
	if strings.HasPrefix(value, "file://") {
		u, err := url.Parse(value)
		if err == nil {
			value = u.Path
		}
	}
	value = strings.ReplaceAll(value, `\ `, " ")
	return strings.TrimSpace(value)
}

func imageMIMEForPath(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".gif":
		return "image/gif", true
	case ".webp":
		return "image/webp", true
	}
	if byExt := mime.TypeByExtension(ext); strings.HasPrefix(byExt, "image/") {
		return byExt, true
	}
	return "", false
}

func isPNG(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' &&
		data[4] == '\r' && data[5] == '\n' && data[6] == 0x1a && data[7] == '\n'
}

func sanitizeName(value string) string {
	var b strings.Builder
	for _, ch := range strings.ToLower(value) {
		switch {
		case ch >= 'a' && ch <= 'z':
			b.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, ErrUnavailable
		}
		return nil, err
	}
	return out, nil
}

func windowsClipboardScript(bitmapPath string) string {
	return `$ErrorActionPreference = 'Stop'; ` +
		`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; ` +
		`if ([System.Windows.Forms.Clipboard]::ContainsFileDropList()) { foreach ($f in [System.Windows.Forms.Clipboard]::GetFileDropList()) { Write-Output ("file" + [char]9 + $f) } }; ` +
		`if ([System.Windows.Forms.Clipboard]::ContainsImage()) { $img = [System.Windows.Forms.Clipboard]::GetImage(); $path = ` + quotePowerShell(bitmapPath) + `; $img.Save($path, [System.Drawing.Imaging.ImageFormat]::Png); Write-Output ("bitmap" + [char]9 + $path) }`
}

func quotePowerShell(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
