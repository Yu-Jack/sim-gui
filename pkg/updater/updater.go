package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type UpdateStatus struct {
	UpdateAvailable bool      `json:"updateAvailable"`
	CurrentCommit   string    `json:"currentCommit"`
	LatestCommit    string    `json:"latestCommit"`
	LastChecked     time.Time `json:"lastChecked"`
	Message         string    `json:"message"`
}

type Updater struct {
	owner      string
	repo       string
	branch     string
	interval   time.Duration
	status     UpdateStatus
	statusLock sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

type GitHubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

func NewUpdater(owner, repo, branch string, interval time.Duration) *Updater {
	ctx, cancel := context.WithCancel(context.Background())
	return &Updater{
		owner:    owner,
		repo:     repo,
		branch:   branch,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
		status: UpdateStatus{
			UpdateAvailable: false,
		},
	}
}

// Start begins checking for updates at the specified interval
func (u *Updater) Start() {
	// Do an initial check
	u.checkForUpdates()

	// If interval is 0, don't schedule periodic checks
	if u.interval == 0 {
		return
	}

	// Start periodic checks
	go func() {
		ticker := time.NewTicker(u.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				u.checkForUpdates()
			case <-u.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the updater
func (u *Updater) Stop() {
	u.cancel()
}

// GetStatus returns the current update status
func (u *Updater) GetStatus() UpdateStatus {
	u.statusLock.RLock()
	defer u.statusLock.RUnlock()
	return u.status
}

// checkForUpdates checks for new commits on GitHub
func (u *Updater) checkForUpdates() {
	currentCommit, err := u.getCurrentCommit()
	if err != nil {
		log.Printf("Failed to get current commit: %v", err)
		u.updateStatus(UpdateStatus{
			UpdateAvailable: false,
			Message:         fmt.Sprintf("Failed to get current commit: %v", err),
			LastChecked:     time.Now(),
		})
		return
	}

	latestCommit, err := u.getLatestCommit()
	if err != nil {
		log.Printf("Failed to get latest commit from GitHub: %v", err)
		u.updateStatus(UpdateStatus{
			UpdateAvailable: false,
			CurrentCommit:   currentCommit,
			Message:         fmt.Sprintf("Failed to check for updates: %v", err),
			LastChecked:     time.Now(),
		})
		return
	}

	updateAvailable := currentCommit != latestCommit
	message := "You are running the latest version"
	if updateAvailable {
		message = "A new update is available! Run 'git pull' to update."
		log.Printf("Update available: current=%s, latest=%s", currentCommit[:7], latestCommit[:7])
	}

	u.updateStatus(UpdateStatus{
		UpdateAvailable: updateAvailable,
		CurrentCommit:   currentCommit,
		LatestCommit:    latestCommit,
		Message:         message,
		LastChecked:     time.Now(),
	})
}

// getCurrentCommit gets the current git commit hash
func (u *Updater) getCurrentCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getLatestCommit fetches the latest commit from GitHub API
func (u *Updater) getLatestCommit() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", u.owner, u.repo, u.branch)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(u.ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to avoid GitHub API restrictions
	req.Header.Set("User-Agent", "sim-gui-updater")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch commit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var commit GitHubCommit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return commit.SHA, nil
}

// updateStatus updates the internal status
func (u *Updater) updateStatus(status UpdateStatus) {
	u.statusLock.Lock()
	defer u.statusLock.Unlock()
	u.status = status
}
