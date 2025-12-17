import React, { useState, useEffect } from 'react';
import { Loader2, CheckCircle2, XCircle, AlertCircle } from 'lucide-react';
import { checkLiveMigration, getNamespaces, getResources, getVirtualMachinePods, type LiveMigrationCheckResult, type PodInfo, type MigrationInfo } from '../../api/client';
import { useToast } from '../../contexts/ToastContext';
import type { Version } from '../../types';

interface Props {
  workspaceName: string;
  versions: Version[];
}

export const LiveMigrationCheck: React.FC<Props> = ({ workspaceName, versions }) => {
  const [selectedVersion, setSelectedVersion] = useState<string>('');
  const [namespace, setNamespace] = useState('');
  const [vmName, setVmName] = useState('');
  const [loading, setLoading] = useState(false);
  const [vmPods, setVmPods] = useState<PodInfo[]>([]);
  const [vmMigrations, setVmMigrations] = useState<MigrationInfo[]>([]);
  const [selectedPod, setSelectedPod] = useState<string | null>(null);
  const [selectedMigrationYaml, setSelectedMigrationYaml] = useState<string | null>(null);
  const [result, setResult] = useState<LiveMigrationCheckResult | null>(null);
  const [availableNamespaces, setAvailableNamespaces] = useState<string[]>([]);
  const [loadingNamespaces, setLoadingNamespaces] = useState(false);
  const [availableVMs, setAvailableVMs] = useState<string[]>([]);
  const [loadingVMs, setLoadingVMs] = useState(false);
  const [showNamespaceSuggestions, setShowNamespaceSuggestions] = useState(false);
  const [showVMSuggestions, setShowVMSuggestions] = useState(false);
  const { showError } = useToast();

  // Set default version to the latest one
  useEffect(() => {
    if (versions.length > 0 && !selectedVersion) {
      setSelectedVersion(versions[versions.length - 1].id);
    }
  }, [versions, selectedVersion]);

  const loadNamespaces = async () => {
    if (!selectedVersion) return;
    setLoadingNamespaces(true);
    try {
      const namespaces = await getNamespaces(workspaceName, selectedVersion);
      setAvailableNamespaces(namespaces);
    } catch (error) {
      console.error('Failed to load namespaces', error);
      showError('Failed to load namespaces');
    } finally {
      setLoadingNamespaces(false);
    }
  };

  // Load namespaces when version changes or component mounts
  useEffect(() => {
    if (selectedVersion) {
        loadNamespaces();
    }
  }, [selectedVersion]);

  const loadVMs = async () => {
    if (!selectedVersion || !namespace) return;
    setLoadingVMs(true);
    try {
        const vms = await getResources(workspaceName, namespace, 'virtualmachines', '', selectedVersion);
        setAvailableVMs(vms || []);
    } catch (error) {
        console.error('Failed to load VMs', error);
        setAvailableVMs([]);
    } finally {
        setLoadingVMs(false);
    }
  };

  useEffect(() => {
      if (namespace) {
          loadVMs();
      } else {
          setAvailableVMs([]);
      }
  }, [namespace, selectedVersion]);


  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedVersion || !namespace || !vmName) return;

    setLoading(true);
    setVmPods([]);
    setVmMigrations([]);
    setSelectedPod(null);
    setSelectedMigrationYaml(null);
    setResult(null);
    try {
      const data = await getVirtualMachinePods(workspaceName, selectedVersion, namespace, vmName);
      if (data.error) {
        showError(data.error);
      } else {
        setVmPods(data.pods);
        setVmMigrations(data.migrations || []);
      }
    } catch (error) {
      console.error('Failed to get VM pods', error);
      const errorMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to get VirtualMachine pods';
      showError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handlePodCheck = async (podName: string) => {
    if (!selectedVersion || !namespace) return;

    setSelectedPod(podName);
    setSelectedMigrationYaml(null);
    setResult(null);
    try {
      const data = await checkLiveMigration(workspaceName, selectedVersion, namespace, podName);
      setResult(data);
    } catch (error) {
      console.error('Failed to check live migration', error);
      const errorMessage = (error as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to check live migration';
      showError(errorMessage);
      setResult(null);
    }
  };

  // Merge pods and migrations by creation time
  const getMergedTimeline = () => {
    type TimelineItem = 
      | { type: 'pod'; data: PodInfo }
      | { type: 'migration'; data: MigrationInfo };

    const items: TimelineItem[] = [
      ...vmPods.map(pod => ({ type: 'pod' as const, data: pod })),
      ...vmMigrations.map(migration => ({ type: 'migration' as const, data: migration }))
    ];

    items.sort((a, b) => {
      const timeA = new Date(a.data.creationTime).getTime();
      const timeB = new Date(b.data.creationTime).getTime();
      return timeB - timeA; // newest first
    });

    return items;
  };

  return (
    <div className="space-y-6">
      <div className="bg-white shadow rounded-lg p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Live Migration Check</h3>
        
        <form onSubmit={handleSearch} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Version
            </label>
            <select
              value={selectedVersion}
              onChange={(e) => setSelectedVersion(e.target.value)}
              className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
            >
              <option value="" disabled>Select a version</option>
              {versions.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.id}
                </option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Namespace
              </label>
              <div className="relative">
                <input
                  type="text"
                  value={namespace}
                  onChange={(e) => setNamespace(e.target.value)}
                  onFocus={() => setShowNamespaceSuggestions(true)}
                  onBlur={() => setTimeout(() => setShowNamespaceSuggestions(false), 200)}
                  placeholder="Select or enter namespace"
                  className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
                />
                {loadingNamespaces && (
                    <div className="absolute right-2 top-2.5">
                        <Loader2 className="h-4 w-4 animate-spin text-gray-400" />
                    </div>
                )}
                {showNamespaceSuggestions && availableNamespaces.length > 0 && (
                  <ul className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none text-sm">
                    {availableNamespaces
                      .filter(ns => ns.toLowerCase().includes(namespace.toLowerCase()))
                      .map(ns => (
                        <li
                          key={ns}
                          className="cursor-pointer select-none relative py-2 pl-3 pr-9 hover:bg-indigo-600 hover:text-white text-gray-900"
                          style={{ fontSize: '14px' }}
                          onClick={() => {
                            setNamespace(ns);
                            setShowNamespaceSuggestions(false);
                          }}
                        >
                          <span className="block truncate">{ns}</span>
                        </li>
                      ))}
                  </ul>
                )}
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                VirtualMachine Name
              </label>
              <div className="relative">
                <input
                    type="text"
                    value={vmName}
                    onChange={(e) => setVmName(e.target.value)}
                    onFocus={() => setShowVMSuggestions(true)}
                    onBlur={() => setTimeout(() => setShowVMSuggestions(false), 200)}
                    placeholder="Select or enter VM name"
                    className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
                />
                 {loadingVMs && (
                    <div className="absolute right-2 top-2.5">
                        <Loader2 className="h-4 w-4 animate-spin text-gray-400" />
                    </div>
                )}
                {showVMSuggestions && availableVMs && availableVMs.length > 0 && (
                  <ul className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none text-sm">
                    {availableVMs
                      .filter(vm => vm.toLowerCase().includes(vmName.toLowerCase()))
                      .map(vm => (
                        <li
                          key={vm}
                          className="cursor-pointer select-none relative py-2 pl-3 pr-9 hover:bg-indigo-600 hover:text-white text-gray-900"
                          style={{ fontSize: '14px' }}
                          onClick={() => {
                            setVmName(vm);
                            setShowVMSuggestions(false);
                          }}
                        >
                          <span className="block truncate">{vm}</span>
                        </li>
                      ))}
                  </ul>
                )}
              </div>
            </div>
          </div>

          <button
            type="submit"
            disabled={loading || !selectedVersion || !namespace || !vmName}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <AlertCircle className="h-4 w-4 mr-2" />}
            {loading ? 'Searching...' : 'Search VirtualMachine'}
          </button>
        </form>
      </div>

      {(vmPods.length > 0 || vmMigrations.length > 0) && (
        <div className="bg-white shadow rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Timeline for VM: {vmName}</h3>
          <div className="space-y-2">
            {getMergedTimeline().map((item, idx) => {
              if (item.type === 'pod') {
                const pod = item.data;
                // Find migrations that have this pod as source or target
                const relatedMigrations = vmMigrations.filter(
                  m => m.sourcePod === pod.name || m.targetPod === pod.name
                );
                
                return (
                  <div key={`pod-${pod.name}-${idx}`}>
                    <button
                      onClick={() => handlePodCheck(pod.name)}
                      className={`w-full text-left px-4 py-3 rounded-lg border transition-colors ${
                        selectedPod === pod.name
                          ? 'border-indigo-500 bg-indigo-50'
                          : 'border-gray-200 hover:border-indigo-300 hover:bg-gray-50'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex flex-col">
                          <span className="font-mono text-sm">{pod.name}</span>
                          <span className="text-xs text-gray-500 mt-1">
                            Created: {new Date(pod.creationTime).toISOString()}
                          </span>
                          {relatedMigrations.length > 0 && (
                            <div className="text-xs text-blue-600 mt-1">
                              {relatedMigrations.map(m => {
                                if (m.sourcePod === pod.name) return `→ Migrated to ${m.targetPod}`;
                                if (m.targetPod === pod.name) return `← Migrated from ${m.sourcePod}`;
                                return null;
                              }).filter(Boolean).join(' • ')}
                            </div>
                          )}
                        </div>
                        {selectedPod === pod.name && (
                          <CheckCircle2 className="h-5 w-5 text-indigo-600" />
                        )}
                      </div>
                    </button>
                  </div>
                );
              } else {
                const migration = item.data;
                return (
                  <div key={`migration-${migration.name}-${idx}`} className="ml-4">
                    <button
                      onClick={() => setSelectedMigrationYaml(migration.yaml)}
                      className={`w-full text-left px-4 py-3 rounded-lg border-2 border-dashed transition-colors ${
                        selectedMigrationYaml === migration.yaml
                          ? 'border-purple-500 bg-purple-50'
                          : 'border-purple-200 hover:border-purple-400 hover:bg-purple-50'
                      }`}
                    >
                      <div className="flex flex-col">
                        <span className="font-mono text-sm text-purple-700">🔄 {migration.name}</span>
                        <span className="text-xs text-gray-500 mt-1">
                          Created: {new Date(migration.creationTime).toISOString()}
                        </span>
                        <span className="text-xs text-purple-600 mt-1">
                          {migration.sourcePod} → {migration.targetPod}
                        </span>
                      </div>
                    </button>
                  </div>
                );
              }
            })}
          </div>
        </div>
      )}

      {selectedMigrationYaml && (
        <div className="bg-white shadow rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Migration YAML</h3>
          <pre className="bg-gray-50 p-4 rounded-lg overflow-x-auto text-xs font-mono">
            {selectedMigrationYaml}
          </pre>
        </div>
      )}

      {result && (
        <div className="bg-white shadow rounded-lg p-6">
          <div className="space-y-4">
            {!result.error && (
              <div className="border-b pb-4">
                <h4 className="text-md font-medium text-gray-900 mb-3">
                  Node-to-Node Compatibility (KubeVirt Labels)
                </h4>
                {(() => {
                    const allNodes = result.nodeResults.map(n => n.nodeName).sort();
                    const incompatibilityMap = new Map<string, Map<string, Array<{key: string, value: string}>>>();
                    result.nodeToNodeCompatibilities?.forEach(item => {
                        if (!incompatibilityMap.has(item.sourceNode)) {
                            incompatibilityMap.set(item.sourceNode, new Map());
                        }
                        incompatibilityMap.get(item.sourceNode)?.set(item.targetNode, item.missingLabels);
                    });
                    
                    const hasIssues = result.nodeToNodeCompatibilities && result.nodeToNodeCompatibilities.length > 0;

                    if (!hasIssues) {
                        return (
                          <div className="bg-green-50 border-l-4 border-green-400 p-4">
                            <div className="flex">
                              <div className="flex-shrink-0">
                                <CheckCircle2 className="h-5 w-5 text-green-400" aria-hidden="true" />
                              </div>
                              <div className="ml-3">
                                <p className="text-sm text-green-700">
                                  All Matched: All nodes are compatible with each other regarding <code className="font-bold">*.node.kubevirt.io/*</code> labels.
                                </p>
                              </div>
                            </div>
                          </div>
                        );
                    }

                    return (
                      <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-gray-200 border">
                          <thead className="bg-gray-50">
                            <tr>
                              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-r bg-gray-100 sticky left-0 z-10">
                                <div className="flex flex-col items-start">
                                  <span>Source ↓ \ Target →</span>
                                </div>
                              </th>
                              {allNodes.map(node => (
                                <th key={node} className="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider border-r min-w-[150px]">
                                  {node}
                                </th>
                              ))}
                            </tr>
                          </thead>
                          <tbody className="bg-white divide-y divide-gray-200">
                            {allNodes.map(source => (
                              <tr key={source}>
                                <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-gray-900 border-r bg-gray-50 sticky left-0 z-10">
                                  {source}
                                </td>
                                {allNodes.map(target => {
                                  if (source === target) {
                                    return (
                                      <td key={target} className="px-4 py-3 text-center text-gray-400 bg-gray-100 border-r">
                                        <span className="text-xl">×</span>
                                      </td>
                                    );
                                  }
                                  const errors = incompatibilityMap.get(source)?.get(target);
                                  if (errors) {
                                    return (
                                      <td key={target} className="px-4 py-3 text-sm text-red-600 border-r bg-red-50 align-top">
                                         <div className="flex flex-col gap-1 max-h-40 overflow-y-auto">
                                            {errors.map((e, i) => (
                                              <div key={i} className="text-xs bg-white border border-red-200 rounded px-2 py-1" title={`${e.key}: ${e.value}`}>
                                                <div className="font-semibold break-all">{e.key}</div>
                                                <div className="text-gray-500 truncate">Missing: {e.value}</div>
                                              </div>
                                            ))}
                                         </div>
                                      </td>
                                    );
                                  }
                                  return (
                                    <td key={target} className="px-4 py-3 text-center text-green-600 border-r">
                                      <CheckCircle2 className="h-6 w-6 mx-auto text-green-500" />
                                    </td>
                                  );
                                })}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    );
                })()}
              </div>
            )}

            {!result.error && (!result.nodeSelector || !Object.keys(result.nodeSelector).includes('kubernetes.io/hostname')) && (
              <div className="bg-yellow-50 border-l-4 border-yellow-400 p-4">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <AlertCircle className="h-5 w-5 text-yellow-400" aria-hidden="true" />
                  </div>
                  <div className="ml-3">
                    <p className="text-sm text-yellow-700">
                      Warning: This pod does not have a <code className="font-bold">kubernetes.io/hostname</code> node selector.
                    </p>
                  </div>
                </div>
              </div>
            )}
            {result.error ? (
              <div className="flex items-center gap-2 text-red-600">
                <XCircle className="h-5 w-5" />
                <span className="font-medium">{result.error}</span>
              </div>
            ) : (
              <>
                <div className="border-b pb-4">
                  <h4 className="text-md font-medium text-gray-900 mb-2">
                    Pod: {result.podName}
                  </h4>
                  {result.nodeSelector && Object.keys(result.nodeSelector).length > 0 ? (
                    <div className="bg-gray-50 rounded p-3 mt-2">
                      <p className="text-sm font-medium text-gray-700 mb-2">Node Selector:</p>
                      <div className="space-y-1">
                        {Object.entries(result.nodeSelector).map(([key, value]) => (
                          <div key={key} className="text-sm font-mono">
                            <span className="text-indigo-600">{key}</span>:{' '}
                            <span className="text-gray-900">{value}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : (
                    <p className="text-sm text-gray-500 italic">No node selector defined</p>
                  )}
                </div>

                <div>
                  <h4 className="text-md font-medium text-gray-900 mb-3">
                    Node Compatibility ({result.nodeResults.length} nodes)
                  </h4>
                  <div className="space-y-3">
                    {result.nodeResults.map((node, idx) => (
                      <div
                        key={idx}
                        className={`border rounded-lg p-4 ${
                          node.matches ? 'border-green-200 bg-green-50' : 'border-red-200 bg-red-50'
                        }`}
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex items-center gap-2">
                            {node.matches ? (
                              <CheckCircle2 className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                            ) : (
                              <XCircle className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
                            )}
                            <div>
                              <p className="font-medium text-gray-900">{node.nodeName}</p>
                              {node.matches ? (
                                <p className="text-sm text-green-700">
                                  ✓ All selectors matched
                                </p>
                              ) : (
                                <div className="mt-2">
                                  <p className="text-sm font-medium text-red-700 mb-1">
                                    Missing labels:
                                  </p>
                                  <div className="space-y-1">
                                    {node.missingLabels.map((label, labelIdx) => (
                                      <div
                                        key={labelIdx}
                                        className="text-sm font-mono bg-white border border-red-200 rounded px-2 py-1 inline-block mr-2"
                                      >
                                        <span className="text-red-700">{label.key}</span>:{' '}
                                        <span className="text-gray-700">{label.value}</span>
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
