package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Yu-Jack/sim-gui/pkg/docker"
	"github.com/Yu-Jack/sim-gui/pkg/executor"
	"github.com/Yu-Jack/sim-gui/pkg/server/model"
)

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
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
	return nil
}

func FindLatestAvailableExecutor(name string, ws *model.Workspace, dockerCli *docker.Client) (executor.Executor, error) {
	for i := len(ws.Versions) - 1; i >= 0; i-- {
		v := ws.Versions[i]
		if v.Type == model.VersionTypeRuntime {
			return executor.NewRuntimeExecutor(v.KubeconfigPath), nil
		}

		iname := fmt.Sprintf("%s-%s", name, v.ID)
		containers, err := dockerCli.FindRunningContainer(iname)
		if err == nil && len(containers) > 0 {
			return executor.NewContainerExecutor(dockerCli, iname), nil
		}
	}
	return nil, fmt.Errorf("no running simulator or runtime cluster found")
}

func ExecKubectl(exec executor.Executor, args ...string) (string, string, error) {
	cmd := append([]string{"kubectl"}, args...)
	env := []string{"KUBECONFIG=/root/.sim/admin.kubeconfig"}
	return exec.Exec(cmd, env)
}
