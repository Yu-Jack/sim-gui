package kubeconfig

import (
	"fmt"
	"os"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// AddContext will attempt to merge the context of the new instance kubeconfig into your existing
// kubeconfig file
func AddContext(fileName string, name, endpoint, port string, contents []byte) error {
	existingContent, err := os.ReadFile(fileName)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing kubeconfig file %s: %w", fileName, err)
	}

	existingConfig := &api.Config{}
	if len(existingContent) > 0 {
		existingConfig, err = clientcmd.Load(existingContent)
		if err != nil {
			return fmt.Errorf("failed to load existing kubeconfig file %s: %w", fileName, err)
		}
	}

	newConfig, err := ConfigureKubeConfig(contents, name, endpoint, port)
	if err != nil {
		return fmt.Errorf("failed to configure kubeconfig for instance %s: %w", name, err)
	}

	mergedConfig := mergeKubeConfig(existingConfig, newConfig)
	return clientcmd.WriteToFile(*mergedConfig, fileName)
}

// ConfigureKubeConfig will massage the data for new instance kubeconfig to make it easier to merge
// and utilize once the kubeconfig's are merged
func ConfigureKubeConfig(contents []byte, name, endpoint, port string) (*api.Config, error) {
	config, err := clientcmd.Load(contents)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// rename user to admin@name
	config.Clusters["default"].Server = fmt.Sprintf("https://%s:%s", endpoint, port)
	newAuthInfoName := fmt.Sprintf("admin@%s", name)
	config.AuthInfos[newAuthInfoName] = config.AuthInfos["default"]
	delete(config.AuthInfos, "default")

	// rename cluster from default to name
	config.Clusters[name] = config.Clusters["default"]
	delete(config.Clusters, "default")

	// rename context from default to clustername
	config.Contexts[name] = config.Contexts["default"]
	delete(config.Contexts, "default")

	// update context with new values for cluster and user
	config.Contexts[name].AuthInfo = newAuthInfoName
	config.Contexts[name].Cluster = name

	// set current-context to new context name
	config.CurrentContext = name
	config.Clusters[name].InsecureSkipTLSVerify = true
	config.Clusters[name].CertificateAuthorityData = nil
	return config, nil
}

// mergeKubeConfig will merge the new config into an existing config, if existing config is empty then the new config is returned
func mergeKubeConfig(existing, new *api.Config) *api.Config {
	// input kubeconfig was empty, so return new config
	if existing == nil {
		return new
	}

	// initialize maps if they are nil
	if existing.Clusters == nil {
		existing.Clusters = make(map[string]*api.Cluster)
	}
	if existing.AuthInfos == nil {
		existing.AuthInfos = make(map[string]*api.AuthInfo)
	}
	if existing.Contexts == nil {
		existing.Contexts = make(map[string]*api.Context)
	}

	// append new config to existing config
	for name, value := range new.Clusters {
		existing.Clusters[name] = value
	}

	for name, value := range new.AuthInfos {
		existing.AuthInfos[name] = value
	}

	for name, value := range new.Contexts {
		existing.Contexts[name] = value
	}
	return existing
}

// RemoveContext is called during instance deletion and will remove the context associated with instanceName from the kubeconfig file
func RemoveContext(fileName, instanceName string) error {
	existingContent, err := os.ReadFile(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			// no fileName found, no further action needed
			return nil
		}
		return fmt.Errorf("failed to read existing kubeconfig file %s: %w", fileName, err)
	}
	config, err := clientcmd.Load(existingContent)
	if err != nil {
		return fmt.Errorf("error loading kubeconfig: %w", err)
	}
	delete(config.Contexts, instanceName)
	delete(config.Clusters, instanceName)
	delete(config.AuthInfos, instanceName)
	return clientcmd.WriteToFile(*config, fileName)
}
