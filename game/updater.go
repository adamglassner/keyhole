package game

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ReleaseInfo struct {
	Version   string
	Tag       string
	Notes     string
	AssetURL  string
	AssetName string
	Size      int64
}

type updState int

const (
	updIdle updState = iota
	updChecking
	updUpToDate
	updAvailable
	updError
	updDownloading
	updReady
)

type UpdateChecker struct {
	mu       sync.Mutex
	state    updState
	latest   *ReleaseInfo
	err      string
	progress float64
	path     string
	manual   bool
}

var updateChecker = &UpdateChecker{}

func (u *UpdateChecker) latestVersion() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.latest == nil {
		return ""
	}
	return u.latest.Version
}

func (u *UpdateChecker) available() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.state == updAvailable && u.latest != nil
}

func (u *UpdateChecker) poll() {
	u.mu.Lock()
	launch := u.state == updReady && u.path != ""
	path := u.path
	if launch {
		u.path = ""
	}
	u.mu.Unlock()
	if launch && runtime.GOOS == "windows" {
		cmd := exec.Command(path)
		if err := cmd.Start(); err == nil {
			QuitRequested = true
		} else {
			u.mu.Lock()
			u.state = updError
			u.err = "Could not start the installer: " + err.Error()
			u.mu.Unlock()
		}
	}
}

// parseVersion turns "v1.2.3" into comparable ints.
func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	var out [3]int
	parts := strings.SplitN(v, ".", 3)
	for i, p := range parts {
		if i > 2 {
			break
		}
		n := ""
		for _, r := range p {
			if r < '0' || r > '9' {
				break
			}
			n += string(r)
		}
		out[i], _ = strconv.Atoi(n)
	}
	return out
}

func newerThan(a, b string) bool {
	x, y := parseVersion(a), parseVersion(b)
	for i := 0; i < 3; i++ {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

func (u *UpdateChecker) check(manual bool) {
	u.mu.Lock()
	if u.state == updChecking || u.state == updDownloading {
		u.mu.Unlock()
		return
	}
	u.state = updChecking
	u.manual = manual
	u.mu.Unlock()
	go func() {
		info, err := fetchLatestRelease()
		u.mu.Lock()
		defer u.mu.Unlock()
		if err != nil {
			u.state = updError
			u.err = err.Error()
			return
		}
		if newerThan(info.Version, Version) {
			u.latest = info
			u.state = updAvailable
		} else {
			u.latest = info
			u.state = updUpToDate
		}
	}()
}

func fetchLatestRelease() (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", AppName+"/"+Version)
	cl := &http.Client{Timeout: 20 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Couldn't reach GitHub. Check your internet connection.")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("No releases have been published on GitHub yet.")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub answered with status %d. Try again later.", resp.StatusCode)
	}
	var r struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("Couldn't read the release information.")
	}
	info := &ReleaseInfo{Version: strings.TrimPrefix(r.TagName, "v"), Tag: r.TagName, Notes: r.Body}
	for _, a := range r.Assets {
		n := strings.ToLower(a.Name)
		if strings.HasSuffix(n, ".exe") && strings.Contains(n, "setup") {
			info.AssetURL, info.AssetName, info.Size = a.URL, a.Name, a.Size
			break
		}
	}
	return info, nil
}

type progressWriter struct {
	w     io.Writer
	total int64
	done  int64
	u     *UpdateChecker
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.done += int64(n)
	if p.total > 0 {
		p.u.mu.Lock()
		p.u.progress = float64(p.done) / float64(p.total)
		p.u.mu.Unlock()
	}
	return n, err
}

func (u *UpdateChecker) download() {
	u.mu.Lock()
	if u.latest == nil || u.state == updDownloading {
		u.mu.Unlock()
		return
	}
	info := u.latest
	if info.AssetURL == "" {
		u.state = updError
		u.err = "This release has no installer attached (expected a file like Keyhole-Setup-" + info.Version + ".exe)."
		u.mu.Unlock()
		return
	}
	u.state = updDownloading
	u.progress = 0
	u.mu.Unlock()
	go func() {
		fail := func(msg string) {
			u.mu.Lock()
			u.state = updError
			u.err = msg
			u.mu.Unlock()
		}
		req, _ := http.NewRequest("GET", info.AssetURL, nil)
		req.Header.Set("User-Agent", AppName+"/"+Version)
		cl := &http.Client{Timeout: 15 * time.Minute}
		resp, err := cl.Do(req)
		if err != nil {
			fail("Download failed. Check your internet connection.")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			fail(fmt.Sprintf("Download failed (status %d).", resp.StatusCode))
			return
		}
		name := info.AssetName
		if name == "" {
			name = "Keyhole-Setup.exe"
		}
		path := filepath.Join(os.TempDir(), name)
		f, err := os.Create(path)
		if err != nil {
			fail("Couldn't save the installer: " + err.Error())
			return
		}
		total := resp.ContentLength
		if total <= 0 {
			total = info.Size
		}
		_, err = io.Copy(&progressWriter{w: f, total: total, u: u}, resp.Body)
		f.Close()
		if err != nil {
			fail("Download was interrupted. Please try again.")
			return
		}
		u.mu.Lock()
		u.progress = 1
		u.state = updReady
		u.path = path
		u.mu.Unlock()
	}()
}

func (u *UpdateChecker) statusText() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	switch u.state {
	case updChecking:
		return "Checking GitHub for updates..."
	case updUpToDate:
		return "You're up to date!"
	case updAvailable:
		return "Version " + u.latest.Version + " is available."
	case updError:
		return u.err
	case updDownloading:
		return fmt.Sprintf("Downloading update... %d%%", int(u.progress*100))
	case updReady:
		if runtime.GOOS == "windows" {
			return "Starting the installer..."
		}
		return "Downloaded. Run the installer from your temp folder."
	}
	return "Updates come from github.com/" + GitHubOwner + "/" + GitHubRepo
}

func (u *UpdateChecker) statusColor() color.RGBA {
	u.mu.Lock()
	defer u.mu.Unlock()
	switch u.state {
	case updUpToDate, updAvailable, updReady:
		return colGood
	case updError:
		return colDanger
	}
	return colTextDim
}
