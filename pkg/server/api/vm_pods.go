package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Yu-Jack/sim-gui/pkg/server/utils"
	"gopkg.in/yaml.v3"
)

type PodInfo struct {
	Name         string `json:"name"`
	CreationTime string `json:"creationTime"`
}

type MigrationInfo struct {
	Name         string `json:"name"`
	CreationTime string `json:"creationTime"`
	SourcePod    string `json:"sourcePod"`
	TargetPod    string `json:"targetPod"`
	Yaml         string `json:"yaml"`
}

type VirtualMachinePodsResult struct {
	VMName     string          `json:"vmName"`
	Pods       []PodInfo       `json:"pods"`
	Migrations []MigrationInfo `json:"migrations"`
	Error      string          `json:"error,omitempty"`
}

type PodList struct {
	Items []struct {
		Metadata struct {
			Name              string            `yaml:"name"`
			Labels            map[string]string `yaml:"labels"`
			CreationTimestamp string            `yaml:"creationTimestamp"`
		} `yaml:"metadata"`
	} `yaml:"items"`
}

type MigrationList struct {
	Items []struct {
		Metadata struct {
			Name              string            `yaml:"name"`
			Labels            map[string]string `yaml:"labels"`
			CreationTimestamp string            `yaml:"creationTimestamp"`
		} `yaml:"metadata"`
		Status struct {
			MigrationState struct {
				SourcePod string `yaml:"sourcePod"`
				TargetPod string `yaml:"targetPod"`
			} `yaml:"migrationState"`
		} `yaml:"status"`
	} `yaml:"items"`
}

func (s *Server) handleGetVMPods(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		VersionID string `json:"versionID"`
		Namespace string `json:"namespace"`
		VMName    string `json:"vmName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.VersionID == "" || req.Namespace == "" || req.VMName == "" {
		http.Error(w, "versionID, namespace and vmName are required", http.StatusBadRequest)
		return
	}

	ws, err := s.store.GetWorkspace(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check if version exists
	if !HasVersionInWorkspace(ws, req.VersionID) {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	instanceName := fmt.Sprintf("%s-%s", name, req.VersionID)

	// Check if container is running
	containers, err := s.docker.FindRunningContainer(instanceName)
	if err != nil || len(containers) == 0 {
		result := VirtualMachinePodsResult{
			VMName: req.VMName,
			Error:  fmt.Sprintf("Simulator for version %s is not running", req.VersionID),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// Check if VM exists
	_, stderr, err := utils.ExecKubectl(s.docker, instanceName, "get", "virtualmachine", req.VMName, "-n", req.Namespace, "-o", "yaml")
	if err != nil || stderr != "" {
		result := VirtualMachinePodsResult{
			VMName: req.VMName,
			Error:  fmt.Sprintf("VirtualMachine '%s' not found in namespace '%s'", req.VMName, req.Namespace),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// Get all pods in namespace with label selector for this VM (including terminated pods)
	// KubeVirt uses labels like kubevirt.io/vm=<vm-name>
	// kubectl get pods returns all pods by default, including Completed/Terminated ones
	podsYAML, stderr, err := utils.ExecKubectl(s.docker, instanceName, "get", "pods", "-n", req.Namespace, "-l", fmt.Sprintf("harvesterhci.io/vmName=%s", req.VMName), "-o", "yaml")
	if err != nil {
		result := VirtualMachinePodsResult{
			VMName: req.VMName,
			Error:  fmt.Sprintf("Failed to get pods for VM: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	if stderr != "" {
		result := VirtualMachinePodsResult{
			VMName: req.VMName,
			Error:  fmt.Sprintf("Failed to list pods: %s", stderr),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	var podList PodList
	if err := yaml.Unmarshal([]byte(podsYAML), &podList); err != nil {
		result := VirtualMachinePodsResult{
			VMName: req.VMName,
			Error:  fmt.Sprintf("Failed to parse pods: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// Extract pod info
	pods := make([]PodInfo, 0)
	for _, pod := range podList.Items {
		if pod.Metadata.Name != "" {
			pods = append(pods, PodInfo{
				Name:         pod.Metadata.Name,
				CreationTime: pod.Metadata.CreationTimestamp,
			})
		}
	}

	// If no pods found with label selector, try matching by prefix (including terminated pods)
	if len(pods) == 0 {
		allPodsYAML, _, err := utils.ExecKubectl(s.docker, instanceName, "get", "pods", "-n", req.Namespace, "-o", "yaml")
		if err == nil {
			var allPodList PodList
			if err := yaml.Unmarshal([]byte(allPodsYAML), &allPodList); err == nil {
				for _, pod := range allPodList.Items {
					if strings.HasPrefix(pod.Metadata.Name, req.VMName+"-") {
						pods = append(pods, PodInfo{
							Name:         pod.Metadata.Name,
							CreationTime: pod.Metadata.CreationTimestamp,
						})
					}
				}
			}
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(pods, func(i, j int) bool {
		ti, errI := time.Parse(time.RFC3339, pods[i].CreationTime)
		tj, errJ := time.Parse(time.RFC3339, pods[j].CreationTime)
		if errI != nil || errJ != nil {
			return false
		}
		return ti.After(tj)
	})

	// Get VirtualMachineInstanceMigrations for this VM
	migrationsYAML, _, err := utils.ExecKubectl(s.docker, instanceName, "get", "virtualmachineinstancemigrations", "-n", req.Namespace, "-l", fmt.Sprintf("kubevirt.io/vmi-name=%s", req.VMName), "-o", "yaml")
	migrations := make([]MigrationInfo, 0)

	if err == nil && migrationsYAML != "" {
		var migrationList MigrationList
		if err := yaml.Unmarshal([]byte(migrationsYAML), &migrationList); err == nil {
			for _, mig := range migrationList.Items {
				if mig.Metadata.Name != "" {
					// Get full YAML for this migration
					migYAML, _, err := utils.ExecKubectl(s.docker, instanceName, "get", "virtualmachineinstancemigration", mig.Metadata.Name, "-n", req.Namespace, "-o", "yaml")
					if err == nil {
						migrations = append(migrations, MigrationInfo{
							Name:         mig.Metadata.Name,
							CreationTime: mig.Metadata.CreationTimestamp,
							SourcePod:    mig.Status.MigrationState.SourcePod,
							TargetPod:    mig.Status.MigrationState.TargetPod,
							Yaml:         migYAML,
						})
					}
				}
			}
		}
	}

	// Sort migrations by creation time (newest first)
	sort.Slice(migrations, func(i, j int) bool {
		ti, errI := time.Parse(time.RFC3339, migrations[i].CreationTime)
		tj, errJ := time.Parse(time.RFC3339, migrations[j].CreationTime)
		if errI != nil || errJ != nil {
			return false
		}
		return ti.After(tj)
	})

	result := VirtualMachinePodsResult{
		VMName:     req.VMName,
		Pods:       pods,
		Migrations: migrations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
