package api

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Yu-Jack/sim-gui/pkg/server/model"
	"github.com/Yu-Jack/sim-gui/pkg/server/utils"
)

func getNextVersionID(ws *model.Workspace) string {
	maxVersion := 0
	for _, v := range ws.Versions {
		var vNum int
		if _, err := fmt.Sscanf(v.ID, "v%d", &vNum); err == nil && vNum > maxVersion {
			maxVersion = vNum
		}
	}
	return fmt.Sprintf("v%d", maxVersion+1)
}

func isKubeconfigFile(files []*multipart.FileHeader) bool {
	if len(files) != 1 {
		return false
	}
	ext := strings.ToLower(filepath.Ext(files[0].Filename))
	return ext == ".kubeconfig" || ext == ".yaml" || ext == ".yml"
}

func processKubeconfigUpload(files []*multipart.FileHeader, versionPath, versionID string) (*model.Version, error) {
	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bundleName := filepath.Base(fileHeader.Filename)
	bundlePath := filepath.Join(versionPath, bundleName)
	destFile, err := os.Create(bundlePath)
	if err != nil {
		return nil, err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		return nil, err
	}

	return &model.Version{
		ID:                versionID,
		Name:              versionID,
		Type:              model.VersionTypeRuntime,
		CreatedAt:         time.Now(),
		KubeconfigPath:    bundlePath,
		Ready:             true,
		SupportBundleName: bundleName,
	}, nil
}

func processSupportBundleUpload(files []*multipart.FileHeader, versionPath, versionID string) (*model.Version, error) {
	var bundlePath string
	var bundleName string

	if len(files) == 1 {
		// Single file
		fileHeader := files[0]
		file, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}
		defer file.Close()

		bundleName = filepath.Base(fileHeader.Filename)
		bundlePath = filepath.Join(versionPath, bundleName)
		destFile, err := os.Create(bundlePath)
		if err != nil {
			return nil, err
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, file); err != nil {
			return nil, err
		}
	} else {
		// Multiple files (split bundle)
		sort.Slice(files, func(i, j int) bool {
			return files[i].Filename < files[j].Filename
		})

		bundleName = "bundle.zip"
		bundlePath = filepath.Join(versionPath, bundleName)

		destFile, err := os.Create(bundlePath)
		if err != nil {
			return nil, err
		}
		defer destFile.Close()

		for _, fileHeader := range files {
			f, err := fileHeader.Open()
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(destFile, f); err != nil {
				f.Close()
				return nil, err
			}
			f.Close()
		}
	}

	// Extract
	extractPath := filepath.Join(versionPath, "extracted")
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		return nil, err
	}

	if err := utils.Unzip(bundlePath, extractPath); err != nil {
		return nil, fmt.Errorf("failed to extract: %v", err)
	}

	return &model.Version{
		ID:                versionID,
		Name:              versionID,
		Type:              model.VersionTypeSupportBundle,
		CreatedAt:         time.Now(),
		SupportBundleName: bundleName,
		BundlePath:        bundlePath,
	}, nil
}
