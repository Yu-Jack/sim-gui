package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"strings"

	"github.com/Yu-Jack/sim-gui/pkg/server/utils"
	"gopkg.in/yaml.v3"
)

type LiveMigrationCheckResult struct {
	PodName                   string                    `json:"podName"`
	NodeSelector              map[string]string         `json:"nodeSelector,omitempty"`
	NodeResults               []NodeCompatibilityResult `json:"nodeResults"`
	NodeToNodeCompatibilities []NodeToNodeCompatibility `json:"nodeToNodeCompatibilities"`
	Error                     string                    `json:"error,omitempty"`
}

type NodeToNodeCompatibility struct {
	SourceNode    string         `json:"sourceNode"`
	TargetNode    string         `json:"targetNode"`
	MissingLabels []MissingLabel `json:"missingLabels"`
}

type NodeCompatibilityResult struct {
	NodeName      string         `json:"nodeName"`
	Matches       bool           `json:"matches"`
	MissingLabels []MissingLabel `json:"missingLabels"`
}

type MissingLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PodSpec struct {
	Spec struct {
		NodeSelector map[string]string `yaml:"nodeSelector"`
	} `yaml:"spec"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
}

type NodeList struct {
	Items []struct {
		Metadata struct {
			Name   string            `yaml:"name"`
			Labels map[string]string `yaml:"labels"`
		} `yaml:"metadata"`
	} `yaml:"items"`
}

func (s *Server) handleCheckLiveMigration(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		VersionID string `json:"versionID"`
		Namespace string `json:"namespace"`
		PodName   string `json:"podName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.VersionID == "" || req.Namespace == "" || req.PodName == "" {
		http.Error(w, "versionID, namespace and podName are required", http.StatusBadRequest)
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
		result := LiveMigrationCheckResult{
			Error: fmt.Sprintf("Simulator for version %s is not running", req.VersionID),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// Get pod spec
	podYAML, stderr, err := utils.ExecKubectl(s.docker, instanceName, "get", "pod", req.PodName, "-n", req.Namespace, "-o", "yaml")
	if err != nil {
		result := LiveMigrationCheckResult{
			Error: fmt.Sprintf("Failed to get pod: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	if stderr != "" {
		result := LiveMigrationCheckResult{
			Error: fmt.Sprintf("Pod not found: %s", stderr),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	var pod PodSpec
	if err := yaml.Unmarshal([]byte(podYAML), &pod); err != nil {
		result := LiveMigrationCheckResult{
			Error: fmt.Sprintf("Failed to parse pod spec: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// Get all nodes
	nodesYAML, stderr, err := utils.ExecKubectl(s.docker, instanceName, "get", "nodes", "-o", "yaml")
	if err != nil {
		result := LiveMigrationCheckResult{
			Error: fmt.Sprintf("Failed to get nodes: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	if stderr != "" {
		result := LiveMigrationCheckResult{
			Error: fmt.Sprintf("Failed to list nodes: %s", stderr),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	var nodeList NodeList
	if err := yaml.Unmarshal([]byte(nodesYAML), &nodeList); err != nil {
		result := LiveMigrationCheckResult{
			Error: fmt.Sprintf("Failed to parse nodes: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	// Check compatibility for each node
	var nodeResults []NodeCompatibilityResult
	for _, node := range nodeList.Items {
		compatibility := checkNodeCompatibility(pod.Spec.NodeSelector, node.Metadata.Labels)
		nodeResults = append(nodeResults, NodeCompatibilityResult{
			NodeName:      node.Metadata.Name,
			Matches:       compatibility.Matches,
			MissingLabels: compatibility.MissingLabels,
		})
	}

	// Check node-to-node compatibility for *.node.kubevirt.io/* labels
	var nodeToNodeResults []NodeToNodeCompatibility
	for _, sourceNode := range nodeList.Items {
		for _, targetNode := range nodeList.Items {
			if sourceNode.Metadata.Name == targetNode.Metadata.Name {
				continue
			}

			var missing []MissingLabel
			for k, v := range sourceNode.Metadata.Labels {
				if strings.Contains(k, "node.kubevirt.io") {
					if targetVal, ok := targetNode.Metadata.Labels[k]; !ok || targetVal != v {
						missing = append(missing, MissingLabel{Key: k, Value: v})
					}
				}
			}

			if len(missing) > 0 {
				nodeToNodeResults = append(nodeToNodeResults, NodeToNodeCompatibility{
					SourceNode:    sourceNode.Metadata.Name,
					TargetNode:    targetNode.Metadata.Name,
					MissingLabels: missing,
				})
			}
		}
	}

	result := LiveMigrationCheckResult{
		PodName:                   pod.Metadata.Name,
		NodeSelector:              pod.Spec.NodeSelector,
		NodeResults:               nodeResults,
		NodeToNodeCompatibilities: nodeToNodeResults,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type CompatibilityCheck struct {
	Matches       bool
	MissingLabels []MissingLabel
}

func checkNodeCompatibility(nodeSelector map[string]string, nodeLabels map[string]string) CompatibilityCheck {
	var missingLabels []MissingLabel

	for key, value := range nodeSelector {
		nodeValue, exists := nodeLabels[key]
		if !exists || nodeValue != value {
			missingLabels = append(missingLabels, MissingLabel{
				Key:   key,
				Value: value,
			})
		}
	}

	return CompatibilityCheck{
		Matches:       len(missingLabels) == 0,
		MissingLabels: missingLabels,
	}
}
