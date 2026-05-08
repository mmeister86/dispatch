package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/matthias/dispatch/internal/provider"
)

const (
	appName     = "dispatch"
	currentFile = "current.json"
	version     = 1
)

type MessageKind string

const (
	KindSystem MessageKind = "system"
	KindUser   MessageKind = "user"
	KindAgent  MessageKind = "agent"
	KindTool   MessageKind = "tool"
)

type Message struct {
	Kind MessageKind `json:"kind"`
	Body string      `json:"body"`
}

type Session struct {
	Version      int                `json:"version"`
	ID           string             `json:"id,omitempty"`
	Title        string             `json:"title,omitempty"`
	CreatedAt    time.Time          `json:"created_at,omitempty"`
	UpdatedAt    time.Time          `json:"updated_at"`
	Messages     []Message          `json:"messages"`
	AgentHistory []provider.Message `json:"agent_history"`
}

type Summary struct {
	ID           string
	Title        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
	Current      bool
}

type Store struct {
	path string
}

func NewStore(path string) Store {
	return Store{path: path}
}

func DefaultPath() (string, error) {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if filepath.IsAbs(stateHome) {
		return filepath.Join(stateHome, appName, "sessions", currentFile), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", appName, "sessions", currentFile), nil
}

func (s Store) Load() (Session, error) {
	path, err := s.resolvePath()
	if err != nil {
		return emptySession(), err
	}

	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return emptySession(), nil
		}
		return emptySession(), fmt.Errorf("load session %s: %w", path, err)
	}

	var sess Session
	if err := json.Unmarshal(body, &sess); err != nil {
		return emptySession(), fmt.Errorf("load session %s: %w", path, err)
	}
	if sess.Version == 0 {
		sess.Version = version
	}
	if sess.ID == "" {
		sess.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if sess.ID == strings.TrimSuffix(currentFile, filepath.Ext(currentFile)) {
			sess.ID = ""
		}
		if sess.ID == "" && hasSessionContent(sess) {
			sess.ID = strings.TrimSuffix(currentFile, filepath.Ext(currentFile))
		}
	}
	return sess, nil
}

func (s Store) Save(sess Session) error {
	path, err := s.resolvePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	sess, err = s.normalizeForSave(sess)
	if err != nil {
		return err
	}
	if err := writeSessionFile(s.sessionPath(sess.ID), sess); err != nil {
		return err
	}
	return writeSessionFile(path, sess)
}

func (s Store) List() ([]Summary, error) {
	path, err := s.resolvePath()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	active, err := s.Load()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if hasSessionContent(active) {
				return []Summary{summaryFromSession(active, true)}, nil
			}
			return nil, nil
		}
		return nil, err
	}

	summaries := make([]Summary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == currentFile || filepath.Ext(entry.Name()) != ".json" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		sess, err := readSessionFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if sess.ID == "" {
			sess.ID = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		}
		summaries = append(summaries, summaryFromSession(sess, active.ID != "" && sess.ID == active.ID))
	}

	if len(summaries) == 0 && hasSessionContent(active) {
		summaries = append(summaries, summaryFromSession(active, true))
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})
	return summaries, nil
}

func (s Store) StartNew() (Session, error) {
	now := time.Now().UTC()
	sess := Session{
		Version:   version,
		ID:        generateID(now),
		Title:     "Neue Session",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Save(sess); err != nil {
		return Session{}, err
	}
	return sess, nil
}

func (s Store) Open(id string) (Session, error) {
	if err := validateID(id); err != nil {
		return Session{}, err
	}
	sess, err := readSessionFile(s.sessionPath(id))
	if err != nil {
		return Session{}, err
	}
	if sess.ID == "" {
		sess.ID = id
	}
	return sess, writeSessionFile(s.mustResolvePath(), sess)
}

func (s Store) Delete(id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	active, err := s.Load()
	if err != nil {
		return err
	}
	if active.ID == id {
		return fmt.Errorf("aktive Session %q kann nicht geloescht werden", id)
	}
	if err := os.Remove(s.sessionPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s Store) Clear() error {
	path, err := s.resolvePath()
	if err != nil {
		return err
	}
	active, _ := s.Load()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if active.ID != "" {
		if err := os.Remove(s.sessionPath(active.ID)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s Store) resolvePath() (string, error) {
	if s.path != "" {
		return s.path, nil
	}
	return DefaultPath()
}

func emptySession() Session {
	return Session{Version: version}
}

func (s Store) mustResolvePath() string {
	path, err := s.resolvePath()
	if err != nil {
		return ""
	}
	return path
}

func (s Store) sessionPath(id string) string {
	path, err := s.resolvePath()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(path), id+".json")
}

func (s Store) normalizeForSave(sess Session) (Session, error) {
	if sess.ID != "" {
		if err := validateID(sess.ID); err != nil {
			return Session{}, err
		}
	}

	now := time.Now().UTC()
	active, _ := s.Load()
	if sess.ID == "" && active.ID != "" {
		sess.ID = active.ID
	}
	if sess.ID == "" {
		sess.ID = generateID(now)
	}
	if sess.CreatedAt.IsZero() {
		if active.ID == sess.ID && !active.CreatedAt.IsZero() {
			sess.CreatedAt = active.CreatedAt
		} else {
			sess.CreatedAt = now
		}
	}
	if strings.TrimSpace(sess.Title) == "" {
		sess.Title = deriveTitle(sess.Messages)
	}
	sess.Version = version
	sess.UpdatedAt = now
	return sess, nil
}

func writeSessionFile(path string, sess Session) error {
	if path == "" {
		return fmt.Errorf("session path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".current-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return os.Chmod(path, 0o600)
}

func readSessionFile(path string) (Session, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return emptySession(), err
	}
	var sess Session
	if err := json.Unmarshal(body, &sess); err != nil {
		return emptySession(), fmt.Errorf("load session %s: %w", path, err)
	}
	if sess.Version == 0 {
		sess.Version = version
	}
	return sess, nil
}

func summaryFromSession(sess Session, current bool) Summary {
	return Summary{
		ID:           sess.ID,
		Title:        displayTitle(sess),
		CreatedAt:    sess.CreatedAt,
		UpdatedAt:    sess.UpdatedAt,
		MessageCount: len(sess.Messages),
		Current:      current,
	}
}

func hasSessionContent(sess Session) bool {
	return sess.ID != "" || len(sess.Messages) > 0 || len(sess.AgentHistory) > 0
}

func displayTitle(sess Session) string {
	if strings.TrimSpace(sess.Title) != "" {
		return sess.Title
	}
	return deriveTitle(sess.Messages)
}

func deriveTitle(messages []Message) string {
	for _, msg := range messages {
		if msg.Kind != KindUser {
			continue
		}
		title := strings.Join(strings.Fields(msg.Body), " ")
		if title == "" {
			continue
		}
		runes := []rune(title)
		if len(runes) > 60 {
			return string(runes[:59]) + "…"
		}
		return title
	}
	return "Neue Session"
}

func generateID(now time.Time) string {
	return now.Format("20060102-150405-000000000")
}

func validateID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("session id fehlt")
	}
	if id != filepath.Base(id) || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("ungueltige session id %q", id)
	}
	return nil
}
