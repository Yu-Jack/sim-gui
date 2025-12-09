import React, { useState, useEffect } from 'react';
import { Loader2, FileText } from 'lucide-react';
import { diffLines } from 'diff';
import { getResourceHistory, getNamespaces, getResourceTypes, getResources, type ResourceHistoryResult } from '../../api/client';
import { useToast } from '../../contexts/ToastContext';

const DiffView: React.FC<{ oldText: string; newText: string }> = ({ oldText, newText }) => {
  const diff = diffLines(oldText, newText);

  return (
    <div className="font-mono text-xs overflow-x-auto bg-white border rounded">
      {diff.map((part, index) => {
        const color = part.added ? 'bg-green-100 text-green-800' : part.removed ? 'bg-red-100 text-red-800' : 'text-gray-600';
        const prefix = part.added ? '+ ' : part.removed ? '- ' : '  ';
        return (
          <div key={index} className={`${color} whitespace-pre-wrap`}>
            {part.value.split('\n').map((line, i) => {
                if (i === part.value.split('\n').length - 1 && line === '') return null; // Skip last empty line from split
                return <div key={i} className="px-2">{prefix}{line}</div>;
            })}
          </div>
        );
      })}
    </div>
  );
};

const ResourceDetailView: React.FC<{
  result: ResourceHistoryResult;
  allResults: ResourceHistoryResult[];
  index: number;
}> = ({ result, allResults, index }) => {
  const [viewMode, setViewMode] = useState<'raw' | 'diff'>(
    allResults.length > 1 && index > 0 ? 'diff' : 'raw'
  );
  const [compareVersionId, setCompareVersionId] = useState<string>(
    index > 0 ? allResults[index - 1].versionID : ''
  );

  const compareResult = allResults.find(r => r.versionID === compareVersionId);

  const diffStats = React.useMemo(() => {
    if (!compareResult || !result.content || !compareResult.content) return null;
    const diff = diffLines(compareResult.content, result.content);
    let added = 0;
    let removed = 0;
    diff.forEach((part) => {
      if (part.added) added += part.count || 0;
      if (part.removed) removed += part.count || 0;
    });
    return { added, removed };
  }, [compareResult, result.content]);

  return (
    <div className="mt-6 border rounded-md overflow-hidden bg-white shadow-sm">
        <div className="px-4 py-3 bg-gray-50 border-b flex justify-between items-center">
            <div className="flex items-center gap-4">
                <h4 className="font-medium text-gray-900">
                    Details for {result.versionID}
                </h4>
                {diffStats && (
                  <span className="text-xs font-medium flex gap-1">
                    <span className="text-green-600">+{diffStats.added}</span>
                    <span className="text-red-600">-{diffStats.removed}</span>
                  </span>
                )}
                <span className={`text-xs px-2 py-1 rounded-full ${
                    result.status === 'found' ? 'bg-green-100 text-green-800' :
                    result.status === 'not_found' ? 'bg-yellow-100 text-yellow-800' :
                    result.status === 'stopped' ? 'bg-gray-100 text-gray-800' :
                    'bg-red-100 text-red-800'
                }`}>
                    {result.status === 'found' ? 'Found' :
                     result.status === 'not_found' ? 'Not Found' :
                     result.status === 'stopped' ? 'Container Stopped' : 'Error'}
                </span>
            </div>
            
            {result.status === 'found' && (
                <div className="flex items-center gap-3">
                    <div className="flex items-center gap-2">
                        <label className="text-xs text-gray-600">Compare with:</label>
                        <select
                            value={compareVersionId}
                            onChange={(e) => {
                                setCompareVersionId(e.target.value);
                                setViewMode('diff');
                            }}
                            className="text-xs border-gray-300 rounded-md shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
                        >
                            <option value="">None</option>
                            {allResults.map((r) => (
                                r.versionID !== result.versionID && (
                                    <option key={r.versionID} value={r.versionID}>
                                        {r.versionID}
                                    </option>
                                )
                            ))}
                        </select>
                    </div>
                    
                    <div className="flex bg-gray-200 rounded-lg p-1">
                        <button
                            onClick={() => setViewMode('raw')}
                            className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                                viewMode === 'raw' ? 'bg-white text-gray-900 shadow' : 'text-gray-600 hover:text-gray-900'
                            }`}
                        >
                            YAML
                        </button>
                        <button
                            onClick={() => setViewMode('diff')}
                            className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                                viewMode === 'diff' ? 'bg-white text-gray-900 shadow' : 'text-gray-600 hover:text-gray-900'
                            }`}
                        >
                            Diff
                        </button>
                    </div>
                </div>
            )}
        </div>

        <div className="p-0">
            {result.status === 'found' ? (
                viewMode === 'diff' && compareResult ? (
                    <DiffView oldText={compareResult.content || ''} newText={result.content} />
                ) : (
                    <pre className="p-4 bg-gray-900 text-gray-100 text-xs overflow-x-auto font-mono">
                        {result.content}
                    </pre>
                )
            ) : (
                <div className="p-8 text-center text-gray-500">
                    {result.error || 'Resource not found in this version.'}
                </div>
            )}
        </div>
    </div>
  );
};

interface ResourceHistoryProps {
  workspaceName: string;
}

export const ResourceHistory: React.FC<ResourceHistoryProps> = ({
  workspaceName,
}) => {
  const [namespace, setNamespace] = useState('');
  const [resourceType, setResourceType] = useState('');
  const [resourceName, setResourceName] = useState('');
  
  const [availableNamespaces, setAvailableNamespaces] = useState<string[]>([]);
  const [availableResourceTypes, setAvailableResourceTypes] = useState<string[]>([]);
  const [resourceNameSuggestions, setResourceNameSuggestions] = useState<string[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [showNamespaceSuggestions, setShowNamespaceSuggestions] = useState(false);
  const [showResourceTypeSuggestions, setShowResourceTypeSuggestions] = useState(false);
  
  const [historyResults, setHistoryResults] = useState<ResourceHistoryResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [selectedVersionId, setSelectedVersionId] = useState<string>('');
  const { showError } = useToast();

  useEffect(() => {
    if (workspaceName) {
      getNamespaces(workspaceName).then(setAvailableNamespaces).catch(console.error);
      getResourceTypes(workspaceName).then(setAvailableResourceTypes).catch(console.error);
    }
  }, [workspaceName]);

  useEffect(() => {
    const timer = setTimeout(async () => {
      if (workspaceName && namespace && resourceType && resourceName) {
        try {
          const suggestions = await getResources(workspaceName, namespace, resourceType, resourceName);
          setResourceNameSuggestions(suggestions || []);
          // Only show suggestions if the input doesn't exactly match one of the suggestions
          // or if there are multiple suggestions
          if (suggestions && suggestions.length > 0) {
             const exactMatch = suggestions.length === 1 && suggestions[0] === resourceName;
             if (!exactMatch) {
                 setShowSuggestions(true);
             }
          }
        } catch (error) {
          console.error('Failed to fetch resource suggestions', error);
        }
      } else {
        setResourceNameSuggestions([]);
        setShowSuggestions(false);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [workspaceName, namespace, resourceType, resourceName]);

  const handleSearch = async () => {
    if (!workspaceName || !resourceType.trim() || !resourceName.trim()) return;
    
    const queryNamespace = namespace.trim() || 'default';
    const query = `${queryNamespace}/${resourceType}/${resourceName}`;
    
    setIsSearching(true);
    try {
      const results = await getResourceHistory(workspaceName, query);
      setHistoryResults(results);
      if (results.length > 0) {
        setSelectedVersionId(results[results.length - 1].versionID);
      } else {
        setSelectedVersionId('');
      }
    } catch (error) {
      console.error('Failed to search resource', error);
      showError('Failed to search resource');
    } finally {
      setIsSearching(false);
    }
  };

  return (
    <div className="bg-white shadow sm:rounded-lg p-6">
      <h3 className="text-lg font-medium leading-6 text-gray-900 mb-4">Resource History Search</h3>
      <div className="space-y-4 mb-6">
        <div className="relative">
          <label className="block text-sm font-medium text-gray-700 mb-1">Namespace</label>
          <input
            type="text"
            value={namespace}
            onChange={(e) => setNamespace(e.target.value)}
            onFocus={() => setShowNamespaceSuggestions(true)}
            onBlur={() => setTimeout(() => setShowNamespaceSuggestions(false), 200)}
            placeholder="default"
            className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
          />
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

        <div className="relative">
          <label className="block text-sm font-medium text-gray-700 mb-1">Resource Type</label>
          <input
            type="text"
            value={resourceType}
            onChange={(e) => setResourceType(e.target.value)}
            onFocus={() => setShowResourceTypeSuggestions(true)}
            onBlur={() => setTimeout(() => setShowResourceTypeSuggestions(false), 200)}
            placeholder="e.g. pods"
            className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
          />
          {showResourceTypeSuggestions && availableResourceTypes.length > 0 && (
            <ul className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none text-sm">
              {availableResourceTypes
                .filter(rt => rt.toLowerCase().includes(resourceType.toLowerCase()))
                .map(rt => (
                  <li
                    key={rt}
                    className="cursor-pointer select-none relative py-2 pl-3 pr-9 hover:bg-indigo-600 hover:text-white text-gray-900"
                    style={{ fontSize: '14px' }}
                    onClick={() => {
                      setResourceType(rt);
                      setShowResourceTypeSuggestions(false);
                    }}
                  >
                    <span className="block truncate">{rt}</span>
                  </li>
                ))}
            </ul>
          )}
        </div>

        <div className="relative">
          <label className="block text-sm font-medium text-gray-700 mb-1">Resource Name</label>
          <input
            type="text"
            value={resourceName}
            onChange={(e) => setResourceName(e.target.value)}
            onFocus={() => {
                if (resourceNameSuggestions.length > 0) setShowSuggestions(true);
            }}
            onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
            placeholder="e.g. my-pod"
            className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          {showSuggestions && resourceNameSuggestions.length > 0 && (
            <ul className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none text-sm">
              {resourceNameSuggestions.map((suggestion) => (
                <li
                  key={suggestion}
                  className="cursor-pointer select-none relative py-2 pl-3 pr-9 hover:bg-indigo-600 hover:text-white text-gray-900"
                  onClick={() => {
                    setResourceName(suggestion);
                    setShowSuggestions(false);
                  }}
                >
                  <span className="block truncate">{suggestion}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <div className="flex justify-end mb-6">
        <button
          onClick={handleSearch}
          disabled={isSearching || !resourceType.trim() || !resourceName.trim()}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 h-[38px]"
        >
          {isSearching ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
          Search
        </button>
      </div>

      {historyResults.length > 0 && (
        <div className="mt-8">
            <div className="overflow-x-auto pb-8">
                <div className="relative min-w-max px-8 pt-4">
                    <div className="absolute left-8 right-8 top-[2.25rem] h-0.5 bg-gray-200 -z-10" />
                    
                    <div className="flex items-start gap-12">
                        {historyResults.map((result) => {
                            const isSelected = selectedVersionId === result.versionID;
                            const isFound = result.status === 'found';
                            
                            return (
                                <div 
                                    key={result.versionID} 
                                    className="flex flex-col items-center gap-2 cursor-pointer group"
                                    onClick={() => setSelectedVersionId(result.versionID)}
                                >
                                    <div className={`
                                        w-14 h-14 rounded-full flex items-center justify-center border-2 transition-all duration-200 bg-white
                                        ${isSelected 
                                            ? 'border-indigo-600 ring-4 ring-indigo-100 scale-110' 
                                            : 'border-gray-300 hover:border-indigo-400'
                                        }
                                    `}>
                                        <FileText className={`w-6 h-6 ${
                                            isFound ? 'text-indigo-600' : 'text-gray-400'
                                        }`} />
                                    </div>
                                    
                                    <div className="text-center">
                                        <div className={`text-sm font-medium ${isSelected ? 'text-indigo-600' : 'text-gray-900'}`}>
                                            {result.versionID}
                                        </div>
                                        <div className={`text-xs font-medium ${
                                            result.status === 'found' ? 'text-green-600' : 'text-yellow-600'
                                        }`}>
                                            {result.status === 'found' ? 'Found' : 'Missing'}
                                        </div>
                                    </div>
                                </div>
                            );
                        })}
                        
                    </div>
                </div>
            </div>

            {selectedVersionId && (
                <ResourceDetailView 
                    key={selectedVersionId}
                    result={historyResults.find(r => r.versionID === selectedVersionId)!}
                    allResults={historyResults}
                    index={historyResults.findIndex(r => r.versionID === selectedVersionId)}
                />
            )}
        </div>
      )}
    </div>
  );
};
