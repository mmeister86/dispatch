package clipboard

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
}

func TestDarwinReaderSavesClipboardBitmapAsPNG(t *testing.T) {
	runner := fakeRunner{
		responses: map[string]fakeResponse{
			"osascript|-e|set output to \"\"|-e|try|-e|set theFiles to the clipboard as «class furl»|-e|repeat with f in theFiles|-e|set output to output & POSIX path of f & linefeed|-e|end repeat|-e|output|-e|on error|-e|\"\"|-e|end try": {},
			"pngpaste|-": {out: tinyPNG},
		},
	}
	reader := Reader{
		GOOS:                "darwin",
		Runner:              runner.run,
		ReadNativeDarwinPNG: func(context.Context) ([]byte, error) { return nil, ErrUnavailable },
		CacheDir:            t.TempDir(),
		Now:                 fixedNow,
	}

	attachments, err := reader.ReadImages(context.Background())
	if err != nil {
		t.Fatalf("ReadImages returned error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("attachments = %#v", attachments)
	}
	got := attachments[0]
	if got.MIMEType != "image/png" || got.Source != "clipboard" || got.OriginalName != "clipboard.png" {
		t.Fatalf("attachment metadata = %#v", got)
	}
	content, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatalf("read saved image: %v", err)
	}
	if !bytes.Equal(content, tinyPNG) {
		t.Fatalf("saved image mismatch: %#v", content)
	}
}

func TestDarwinReaderReportsUnavailableWhenBitmapExistsButCannotBeRead(t *testing.T) {
	runner := fakeRunner{
		responses: map[string]fakeResponse{
			"osascript|-e|set output to \"\"|-e|try|-e|set theFiles to the clipboard as «class furl»|-e|repeat with f in theFiles|-e|set output to output & POSIX path of f & linefeed|-e|end repeat|-e|output|-e|on error|-e|\"\"|-e|end try": {},
			"pngpaste|-": {err: ErrUnavailable},
			"osascript|-e|set pngData to the clipboard as «class PNGf»": {},
		},
	}
	reader := Reader{
		GOOS:                "darwin",
		Runner:              runner.run,
		ReadNativeDarwinPNG: func(context.Context) ([]byte, error) { return nil, ErrUnavailable },
		CacheDir:            t.TempDir(),
		Now:                 fixedNow,
	}

	_, err := reader.ReadImages(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if strings.Contains(err.Error(), "pngpaste") {
		t.Fatalf("error should not require pngpaste: %v", err)
	}
}

func TestDarwinReaderUsesBundledBitmapReaderWhenPNGPasteIsMissing(t *testing.T) {
	runner := fakeRunner{
		responses: map[string]fakeResponse{
			"osascript|-e|set output to \"\"|-e|try|-e|set theFiles to the clipboard as «class furl»|-e|repeat with f in theFiles|-e|set output to output & POSIX path of f & linefeed|-e|end repeat|-e|output|-e|on error|-e|\"\"|-e|end try": {},
			"pngpaste|-": {err: ErrUnavailable},
		},
	}
	reader := Reader{
		GOOS:                "darwin",
		Runner:              runner.run,
		ReadNativeDarwinPNG: func(context.Context) ([]byte, error) { return tinyPNG, nil },
		CacheDir:            t.TempDir(),
		Now:                 fixedNow,
	}

	attachments, err := reader.ReadImages(context.Background())
	if err != nil {
		t.Fatalf("ReadImages returned error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("attachments = %#v", attachments)
	}
	got := attachments[0]
	if got.Source != "clipboard" || got.MIMEType != "image/png" || got.OriginalName != "clipboard.png" {
		t.Fatalf("attachment metadata = %#v", got)
	}
	content, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatalf("read saved image: %v", err)
	}
	if !bytes.Equal(content, tinyPNG) {
		t.Fatalf("saved image mismatch: %#v", content)
	}
}

func TestLinuxReaderCopiesMultipleImageFilesFromURIList(t *testing.T) {
	sourceDir := t.TempDir()
	first := filepath.Join(sourceDir, "first.png")
	second := filepath.Join(sourceDir, "second.jpg")
	if err := os.WriteFile(first, tinyPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, append([]byte{0xff, 0xd8, 0xff, 0xdb}, bytes.Repeat([]byte{0}, 16)...), 0o600); err != nil {
		t.Fatal(err)
	}

	uriList := "file://" + first + "\n# comment\nfile://" + second + "\n"
	runner := fakeRunner{
		responses: map[string]fakeResponse{
			"wl-paste|--type|text/uri-list": {out: []byte(uriList)},
			"wl-paste|--type|image/png":     {err: ErrNoImages},
		},
	}
	reader := Reader{
		GOOS:     "linux",
		Runner:   runner.run,
		CacheDir: t.TempDir(),
		Now:      fixedNow,
	}

	attachments, err := reader.ReadImages(context.Background())
	if err != nil {
		t.Fatalf("ReadImages returned error: %v", err)
	}
	if len(attachments) != 2 {
		t.Fatalf("attachments = %#v", attachments)
	}
	for _, attachment := range attachments {
		if !strings.HasPrefix(attachment.Path, reader.CacheDir) {
			t.Fatalf("attachment should be copied into cache dir: %#v", attachment)
		}
		if attachment.Source != "file" {
			t.Fatalf("source = %q, want file", attachment.Source)
		}
	}
}

func TestAttachPastedImagePathsCopiesImageFilesFromBracketedPasteText(t *testing.T) {
	sourceDir := t.TempDir()
	first := filepath.Join(sourceDir, "first image.png")
	second := filepath.Join(sourceDir, "second.jpg")
	if err := os.WriteFile(first, tinyPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, append([]byte{0xff, 0xd8, 0xff, 0xdb}, bytes.Repeat([]byte{0}, 16)...), 0o600); err != nil {
		t.Fatal(err)
	}

	reader := Reader{CacheDir: t.TempDir(), Now: fixedNow}
	pasted := "file://" + strings.ReplaceAll(first, " ", "%20") + "\n" + strings.ReplaceAll(second, " ", `\ `)
	attachments, err := reader.AttachPastedImagePaths(context.Background(), pasted)
	if err != nil {
		t.Fatalf("AttachPastedImagePaths returned error: %v", err)
	}
	if len(attachments) != 2 {
		t.Fatalf("attachments = %#v", attachments)
	}
	if attachments[0].OriginalName != "first image.png" || attachments[1].OriginalName != "second.jpg" {
		t.Fatalf("attachment names = %#v", attachments)
	}
}

func TestWindowsReaderUsesPowerShellClipboardOutput(t *testing.T) {
	source := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(source, tinyPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	bitmapDir := t.TempDir()
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "powershell" || len(args) != 4 || !strings.Contains(args[3], "ContainsImage") {
			return nil, ErrNoImages
		}
		bitmap := filepath.Join(bitmapDir, "clipboard.png")
		if err := os.WriteFile(bitmap, tinyPNG, 0o600); err != nil {
			return nil, err
		}
		return []byte("file\t" + source + "\nbitmap\t" + bitmap + "\n"), nil
	}
	reader := Reader{
		GOOS:     "windows",
		Runner:   runner,
		CacheDir: t.TempDir(),
		Now:      fixedNow,
	}

	attachments, err := reader.ReadImages(context.Background())
	if err != nil {
		t.Fatalf("ReadImages returned error: %v", err)
	}
	if len(attachments) != 2 {
		t.Fatalf("attachments = %#v", attachments)
	}
	if attachments[0].Source != "file" || attachments[1].Source != "clipboard" {
		t.Fatalf("sources = %#v", attachments)
	}
}

func TestLinuxReaderReportsMissingClipboardUtilities(t *testing.T) {
	reader := Reader{
		GOOS:     "linux",
		Runner:   func(context.Context, string, ...string) ([]byte, error) { return nil, ErrUnavailable },
		CacheDir: t.TempDir(),
		Now:      fixedNow,
	}

	_, err := reader.ReadImages(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if !strings.Contains(err.Error(), "wl-paste") || !strings.Contains(err.Error(), "xclip") {
		t.Fatalf("error should name missing tools: %v", err)
	}
}

type fakeResponse struct {
	out []byte
	err error
}

type fakeRunner struct {
	responses map[string]fakeResponse
}

func (f fakeRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	for _, arg := range args {
		key += "|" + arg
	}
	response, ok := f.responses[key]
	if !ok {
		return nil, ErrNoImages
	}
	return response.out, response.err
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 15, 12, 30, 0, 0, time.UTC)
}
