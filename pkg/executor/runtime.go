package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

type RuntimeExecutor struct {
	kubeconfigPath string
}

func NewRuntimeExecutor(kubeconfigPath string) *RuntimeExecutor {
	return &RuntimeExecutor{
		kubeconfigPath: kubeconfigPath,
	}
}

func (e *RuntimeExecutor) Exec(command []string, env []string) (string, string, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", e.kubeconfigPath))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), stderr.String(), nil
}
