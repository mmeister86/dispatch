package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	UpdatedAt    time.Time          `json:"updated_at"`
	Messages     []Message          `json:"messages"`
	AgentHistory []provider.Message `json:"agent_history"`
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

	sess.Version = version
	sess.UpdatedAt = time.Now().UTC()
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

func (s Store) Clear() error {
	path, err := s.resolvePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
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
