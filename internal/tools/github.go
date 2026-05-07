package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/matthias/dispatch/internal/provider"
)

type GitHub struct {
	token   string
	baseURL string
	client  *http.Client
}

func NewGitHub(token, baseURL string) *GitHub {
	return &GitHub{token: token, baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 30 * time.Second}}
}

func (g *GitHub) Definitions() []provider.ToolDefinition {
	return []provider.ToolDefinition{
		{Name: "get_recent_commits", Description: "Holt aktuelle Commits aus einem oder allen GitHub-Repos.", Schema: objectSchema(map[string]any{
			"repos": stringArrayProp(`Repo-Namen oder ["all"] fuer alle`),
			"since": stringProp(`Zeitraum: "24h", "7d" oder "30d"`),
			"limit": intProp("Maximale Commits pro Repo"),
		}, "repos")},
		{Name: "get_activity", Description: "Holt Commits, PRs und Issues kombiniert aus GitHub.", Schema: objectSchema(map[string]any{
			"repos":   stringArrayProp(`Repo-Namen oder ["all"] fuer alle`),
			"since":   stringProp(`Zeitraum: "24h", "7d" oder "30d"`),
			"include": stringArrayProp(`Eine Auswahl aus "commits", "prs", "issues"`),
		}, "repos")},
		{Name: "get_file", Description: "Liest eine Datei aus einem GitHub-Repo.", Schema: objectSchema(map[string]any{
			"repo": stringProp("Repo als owner/name oder Name aus deinen Repos"),
			"path": stringProp("Dateipfad, z.B. README.md"),
		}, "repo", "path")},
	}
}

func (g *GitHub) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	if g.token == "" {
		return "", fmt.Errorf("GitHub Token fehlt")
	}
	switch name {
	case "get_recent_commits":
		var req struct {
			Repos []string `json:"repos"`
			Since string   `json:"since"`
			Limit int      `json:"limit"`
		}
		if err := json.Unmarshal(args, &req); err != nil {
			return "", err
		}
		if req.Limit <= 0 {
			req.Limit = 5
		}
		return g.recentCommits(ctx, req.Repos, req.Since, req.Limit)
	case "get_activity":
		var req struct {
			Repos   []string `json:"repos"`
			Since   string   `json:"since"`
			Include []string `json:"include"`
		}
		if err := json.Unmarshal(args, &req); err != nil {
			return "", err
		}
		return g.activity(ctx, req.Repos, req.Since, req.Include)
	case "get_file":
		var req struct {
			Repo string `json:"repo"`
			Path string `json:"path"`
		}
		if err := json.Unmarshal(args, &req); err != nil {
			return "", err
		}
		return g.file(ctx, req.Repo, req.Path)
	default:
		return "", fmt.Errorf("GitHub Tool %q unbekannt", name)
	}
}

type githubRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Stars    int    `json:"stargazers_count"`
}

type githubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type githubAPIError struct {
	statusCode int
	status     string
	body       string
	message    string
}

func (e githubAPIError) Error() string {
	return fmt.Sprintf("GitHub API returned %s: %s", e.status, e.body)
}

func (g *GitHub) recentCommits(ctx context.Context, repos []string, since string, limit int) (string, error) {
	fullNames, err := g.resolveRepos(ctx, repos)
	if err != nil {
		return "", err
	}
	sinceTime := parseSince(since)
	var out []string
	for _, repo := range fullNames {
		var commits []githubCommit
		q := url.Values{}
		q.Set("per_page", fmt.Sprint(limit))
		if !sinceTime.IsZero() {
			q.Set("since", sinceTime.Format(time.RFC3339))
		}
		if err := g.get(ctx, "/repos/"+repo+"/commits?"+q.Encode(), &commits); err != nil {
			if isEmptyGitRepository(err) {
				out = append(out, "Repo: "+repo)
				out = append(out, "- Keine Commits vorhanden.")
				continue
			}
			return "", err
		}
		out = append(out, "Repo: "+repo)
		for _, c := range commits {
			msg := strings.Split(c.Commit.Message, "\n")[0]
			out = append(out, fmt.Sprintf("- %s %s (%s)", shortSHA(c.SHA), msg, c.Commit.Author.Date.Format("2006-01-02")))
		}
	}
	return strings.Join(out, "\n"), nil
}

func (g *GitHub) activity(ctx context.Context, repos []string, since string, include []string) (string, error) {
	commits, err := g.recentCommits(ctx, repos, since, 5)
	if err != nil {
		return "", err
	}
	return commits + "\n\nPRs und Issues werden ueber die GitHub-Endpunkte vorbereitet; nutze get_recent_commits fuer den aktuellen Durchstich.", nil
}

func (g *GitHub) file(ctx context.Context, repo, path string) (string, error) {
	if !strings.Contains(repo, "/") {
		repos, err := g.resolveRepos(ctx, []string{repo})
		if err != nil {
			return "", err
		}
		if len(repos) == 0 {
			return "", fmt.Errorf("Repo %q nicht gefunden", repo)
		}
		repo = repos[0]
	}
	var res struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := g.get(ctx, "/repos/"+repo+"/contents/"+url.PathEscape(path), &res); err != nil {
		return "", err
	}
	if res.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(res.Content, "\n", ""))
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}
	return res.Content, nil
}

func (g *GitHub) resolveRepos(ctx context.Context, requested []string) ([]string, error) {
	if len(requested) == 0 || (len(requested) == 1 && requested[0] == "all") {
		var repos []githubRepo
		if err := g.get(ctx, "/user/repos?per_page=100&sort=updated", &repos); err != nil {
			return nil, err
		}
		fullNames := make([]string, 0, len(repos))
		for _, repo := range repos {
			fullNames = append(fullNames, repo.FullName)
		}
		return fullNames, nil
	}
	var all []githubRepo
	_ = g.get(ctx, "/user/repos?per_page=100&sort=updated", &all)
	var fullNames []string
	for _, wanted := range requested {
		if strings.Contains(wanted, "/") {
			fullNames = append(fullNames, wanted)
			continue
		}
		for _, repo := range all {
			if repo.Name == wanted {
				fullNames = append(fullNames, repo.FullName)
			}
		}
	}
	return fullNames, nil
}

func (g *GitHub) get(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/vnd.github+json")
	req.Header.Set("authorization", "Bearer "+g.token)
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		bodyText := strings.TrimSpace(string(body))
		apiErr := githubAPIError{statusCode: resp.StatusCode, status: resp.Status, body: bodyText}
		var payload struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			apiErr.message = payload.Message
		}
		return apiErr
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func isEmptyGitRepository(err error) bool {
	apiErr, ok := err.(githubAPIError)
	if !ok || apiErr.statusCode != http.StatusConflict {
		return false
	}
	return strings.EqualFold(strings.TrimSuffix(apiErr.message, "."), "Git Repository is empty")
}

func parseSince(value string) time.Time {
	now := time.Now().UTC()
	switch value {
	case "24h":
		return now.Add(-24 * time.Hour)
	case "30d":
		return now.AddDate(0, 0, -30)
	default:
		return now.AddDate(0, 0, -7)
	}
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}
