package docker

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultBundleDir = "bundle"
)

type TarHandler struct {
	TmpDirName string
}

func NewTarHandler() (*TarHandler, error) {
	dirName, err := os.MkdirTemp(os.TempDir(), "sim-cli")
	if err != nil {
		return nil, fmt.Errorf("error creating temp dir: %v", err)
	}
	return &TarHandler{
		TmpDirName: dirName,
	}, nil
}

func (t *TarHandler) Cleanup() error {
	return os.RemoveAll(t.TmpDirName)
}

// UnzipSupportBundle will unzip bundle into memory FS
// which can then be used to generate a tar ball for providing a context
// to build image with bundle packaged into support-bundle-kit base image
func (t *TarHandler) UnzipSupportBundle(bundleZipFile string) (err error) {
	// Create a temporary directory for extraction
	extractDir, err := os.MkdirTemp("", "sim-cli-extract")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)

	r, err := zip.OpenReader(bundleZipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(extractDir, f.Name)

		// Check for ZipSlip
		if !strings.HasPrefix(fpath, filepath.Clean(extractDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	// Analyze structure to determine the root of the bundle
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return err
	}

	var validEntries []os.DirEntry
	for _, e := range entries {
		if e.Name() == "__MACOSX" || e.Name() == ".DS_Store" {
			continue
		}
		validEntries = append(validEntries, e)
	}

	targetBundlePath := filepath.Join(t.TmpDirName, defaultBundleDir)

	// If there is exactly one directory, assume it is the root folder of the bundle
	if len(validEntries) == 1 && validEntries[0].IsDir() {
		src := filepath.Join(extractDir, validEntries[0].Name())
		return os.Rename(src, targetBundlePath)
	}

	// Otherwise, assume the zip contents are the bundle contents (flat structure or multiple roots)
	// We rename the extraction directory itself to the target bundle path
	return os.Rename(extractDir, targetBundlePath)
}

// GenerateBundleTar attempts to parse FS/bundle to build a tar which can be passed
// as context for image creation step
func (t *TarHandler) GenerateBundleTar() (*bytes.Buffer, error) {
	fs := os.DirFS(t.TmpDirName)
	buf := &bytes.Buffer{}
	contextTar := tar.NewWriter(buf)
	if err := contextTar.AddFS(fs); err != nil {
		return nil, fmt.Errorf("error adding tmp directory %s to tar: %v", t.TmpDirName, err)
	}

	if err := contextTar.Close(); err != nil {
		return nil, fmt.Errorf("error closing tar file %v", err)
	}
	return buf, nil
}

func (t *TarHandler) AddDockerFile(baseImage string) error {
	dockerFile, err := generateTemplate(baseImage)
	if err != nil {
		return fmt.Errorf("error generating dockerfile from embedded template")
	}

	return os.WriteFile(filepath.Join(t.TmpDirName, "Dockerfile"), dockerFile.Bytes(), 0700)
}

// ReadTar is a helper utility to read contents of in memory tar file
// and is mostly used for debugging and testing
func ReadTar(buf *bytes.Buffer) error {
	tr := tar.NewReader(buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		fmt.Printf("Contents of %s:\n", hdr.Name)
	}
	return nil
}

// BuildContextTar is a wrapper function tht builds a tar ball with Dockerfile and contents of bundle
// and this can be passed to image builder to ensure support bundle kit image is layered with
// actual support bundle contents to allow for subsequent processing by simulator
func BuildContextTar(bundlePath string, baseImage string) (*bytes.Buffer, error) {
	t, err := NewTarHandler()
	if err != nil {
		return nil, err
	}

	// prepare zip file and extract it into bundle folder
	if err := t.UnzipSupportBundle(bundlePath); err != nil {
		return nil, err
	}

	// add Dockerfile to root of tar image
	if err := t.AddDockerFile(baseImage); err != nil {
		return nil, err
	}

	buf, err := t.GenerateBundleTar()
	if err != nil {
		return nil, err
	}

	if err := t.Cleanup(); err != nil {
		return nil, fmt.Errorf("error cleaning up tmp directory: %v", err)
	}
	return buf, err
}

func generateTemplate(baseImage string) (bytes.Buffer, error) {
	contents := struct {
		BaseImage string
	}{
		BaseImage: baseImage,
	}

	dockerFile := `FROM {{ .BaseImage }}
EXPOSE 6443/tcp
COPY bundle /bundle
`
	dockerTemplate := template.Must(template.New("dockerfile").Parse(dockerFile))
	var output bytes.Buffer
	err := dockerTemplate.Execute(&output, contents)
	return output, err
}
