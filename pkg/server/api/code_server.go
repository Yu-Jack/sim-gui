package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func recursiveExtract(root string) error {
	for {
		var archives []string
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "__MACOSX" {
					return filepath.SkipDir
				}
				return nil
			}

			name := info.Name()
			if strings.HasPrefix(name, "._") {
				return nil
			}

			if strings.HasSuffix(name, ".zip") ||
				strings.HasSuffix(name, ".tar.gz") ||
				strings.HasSuffix(name, ".tgz") ||
				strings.HasSuffix(name, ".tar.xz") ||
				strings.HasSuffix(name, ".txz") ||
				strings.HasSuffix(name, ".tar") {
				archives = append(archives, path)
			}
			return nil
		})
		if err != nil {
			return err
		}

		if len(archives) == 0 {
			break
		}

		for _, archive := range archives {
			dir := filepath.Dir(archive)
			name := filepath.Base(archive)
			var cmd *exec.Cmd

			if strings.HasSuffix(name, ".zip") {
				cmd = exec.Command("unzip", "-q", "-o", archive, "-d", dir, "-x", "__MACOSX/*", "*/__MACOSX/*", "._*")
			} else if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") {
				cmd = exec.Command("tar", "--exclude=__MACOSX", "--exclude=._*", "-xzf", archive, "-C", dir)
			} else if strings.HasSuffix(name, ".tar.xz") || strings.HasSuffix(name, ".txz") {
				cmd = exec.Command("tar", "--exclude=__MACOSX", "--exclude=._*", "-xJf", archive, "-C", dir)
			} else if strings.HasSuffix(name, ".tar") {
				cmd = exec.Command("tar", "--exclude=__MACOSX", "--exclude=._*", "-xf", archive, "-C", dir)
			}

			if cmd != nil {
				if output, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("failed to extract %s: %v, output: %s", archive, err, string(output))
				}

				// Fix permissions after extraction to ensure we can walk/remove
				// Some archives might contain read-only directories which causes filepath.Walk to fail
				exec.Command("chmod", "-R", "755", dir).Run()

				if err := os.Remove(archive); err != nil {
					return fmt.Errorf("failed to remove %s: %v", archive, err)
				}
			}
		}
	}
	return nil
}

func (s *Server) handleStartCodeServer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	versionID := r.PathValue("versionID")

	// Find bundle file
	versionPath := filepath.Join(s.dataDir, "workspaces", name, versionID)
	entries, err := os.ReadDir(versionPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var bundlePath string
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != "extracted" {
			bundlePath = filepath.Join(versionPath, entry.Name())
			break
		}
	}

	if bundlePath == "" {
		http.Error(w, "Bundle file not found", http.StatusNotFound)
		return
	}

	instanceName := "sim-cli-code-server"

	url, _, err := s.docker.RunCodeServer(instanceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if directory already exists in container
	targetDir := fmt.Sprintf("/home/coder/project/%s-%s", name, versionID)
	if _, _, err := s.docker.ExecContainer(instanceName, []string{"test", "-d", targetDir}, nil); err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"url": url,
		})
		return
	}

	// Prepare temp directory for extraction
	tempRoot, err := os.MkdirTemp("", "sim-cli-extract")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// defer os.RemoveAll(tempRoot)

	extractDirName := fmt.Sprintf("%s-%s", name, versionID)
	extractDirPath := filepath.Join(tempRoot, extractDirName)
	if err := os.Mkdir(extractDirPath, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy bundle to temp dir
	srcFile, err := os.Open(bundlePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer srcFile.Close()

	destBundlePath := filepath.Join(extractDirPath, filepath.Base(bundlePath))
	destFile, err := os.Create(destBundlePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// We close explicitly later, but defer just in case
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	destFile.Close()

	// Recursive extract
	if err := recursiveExtract(extractDirPath); err != nil {
		http.Error(w, fmt.Sprintf("Extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Fix permissions on host before copying
	// We use chmod -R 755 to ensure directories are accessible and files are readable
	cmdChmod := exec.Command("chmod", "-R", "755", tempRoot)
	if output, err := cmdChmod.CombinedOutput(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to chmod extracted files: %v, output: %s", err, string(output)), http.StatusInternalServerError)
		return
	}

	// Ensure parent directory exists in container
	_, _, err = s.docker.ExecContainer(instanceName, []string{"mkdir", "-p", "/home/coder/project"}, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy extracted directory to container
	cmdCp := exec.Command("docker", "cp", extractDirPath, fmt.Sprintf("%s:/home/coder/project/", instanceName))
	if output, err := cmdCp.CombinedOutput(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to copy files via docker cp: %v, output: %s", err, string(output)), http.StatusInternalServerError)
		return
	}

	// Fix permissions
	_, _, err = s.docker.ExecContainer(instanceName, []string{"sudo", "chown", "coder:coder", "-R", "/home/coder/project"}, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fix permissions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": url,
	})
}
