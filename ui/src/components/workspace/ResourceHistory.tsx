import React, { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { diffLines } from 'diff';
import { getResourceHistory, getNamespaces, getResourceTypes, getResources, type ResourceHistoryResult } from '../../api/client';

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

const ResourceHistoryCard: React.FC<{
  result: ResourceHistoryResult;
  allResults: ResourceHistoryResult[];
  index: number;
}> = ({ result, allResults, index }) => {
  const [viewMode, setViewMode] = useState<'collapsed' | 'raw' | 'diff'>('collapsed');
  const [compareVersionId, setCompareVersionId] = useState<string>(
    index > 0 ? allResults[index - 1].versionID : ''
  );

  const compareResult = allResults.find(r => r.versionID === compareVersionId);

  return (
    <div className="border rounded-md overflow-hidden">
      <div className={`px-4 py-2 bg-gray-50 border-b flex justify-between items-center ${
        result.status === 'found' ? 'border-green-200 bg-green-50' :
        result.status === 'not_found' ? 'border-yellow-200 bg-yellow-50' :
        'border-gray-200'
      }`}>
        <div className="flex items-center gap-4">
          <span className="font-medium text-sm text-gray-700">Version: {result.versionID}</span>
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
          <div className="flex items-center gap-2">
            {viewMode === 'diff' && (
              <select
                value={compareVersionId}
                onChange={(e) => setCompareVersionId(e.target.value)}
                className="text-xs border-gray-300 rounded-md shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
              >
                <option value="" disabled>Compare with...</option>
                {allResults.map((r) => (
                  r.versionID !== result.versionID && (
                    <option key={r.versionID} value={r.versionID}>
                      {r.versionID}
                    </option>
                  )
                ))}
              </select>
            )}
            {viewMode === 'collapsed' ? (
              <>
                <button
                  onClick={() => setViewMode('raw')}
                  className="text-xs text-gray-600 hover:text-gray-900 font-medium"
                >
                  Show YAML
                </button>
                <button
                  onClick={() => setViewMode('diff')}
                  className="text-xs text-indigo-600 hover:text-indigo-900 font-medium"
                >
                  Show Diff
                </button>
              </>
            ) : (
              <>
                <button
                  onClick={() => setViewMode(viewMode === 'raw' ? 'diff' : 'raw')}
                  className="text-xs text-indigo-600 hover:text-indigo-900 font-medium"
                >
                  {viewMode === 'raw' ? 'Show Diff' : 'Show YAML'}
                </button>
                <button
                  onClick={() => setViewMode('collapsed')}
                  className="text-xs text-gray-600 hover:text-gray-900 font-medium"
                >
                  Hide
                </button>
              </>
            )}
          </div>
        )}
      </div>

      {result.status === 'found' && viewMode !== 'collapsed' && (
        <div className="p-0">
          {viewMode === 'diff' && compareResult ? (
            <DiffView oldText={compareResult.content || ''} newText={result.content} />
          ) : (
            <pre className="p-4 bg-gray-900 text-gray-100 text-xs overflow-x-auto font-mono">
              {result.content}
            </pre>
          )}
        </div>
      )}
      {result.error && (
        <div className="p-4 text-sm text-red-600 bg-red-50">
          {result.error}
        </div>
      )}
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
  
  const [historyResults, setHistoryResults] = useState<ResourceHistoryResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);

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
          setShowSuggestions(true);
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
    } catch (error) {
      console.error('Failed to search resource', error);
      alert('Failed to search resource');
    } finally {
      setIsSearching(false);
    }
  };

  return (
    <div className="bg-white shadow sm:rounded-lg p-6">
      <h3 className="text-lg font-medium leading-6 text-gray-900 mb-4">Resource History Search</h3>
      <div className="space-y-4 mb-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Namespace</label>
          <input
            type="text"
            list="namespaces"
            value={namespace}
            onChange={(e) => setNamespace(e.target.value)}
            placeholder="default"
            className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
          />
          <datalist id="namespaces">
            {availableNamespaces.map(ns => <option key={ns} value={ns} />)}
          </datalist>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Resource Type</label>
          <input
            type="text"
            list="resourceTypes"
            value={resourceType}
            onChange={(e) => setResourceType(e.target.value)}
            placeholder="e.g. pods"
            className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
          />
          <datalist id="resourceTypes">
            {availableResourceTypes.map(rt => <option key={rt} value={rt} />)}
          </datalist>
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
            <ul className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm">
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
        <div className="space-y-4">
          {historyResults.map((result, index) => (
            <ResourceHistoryCard key={result.versionID} result={result} allResults={historyResults} index={index} />
          ))}
        </div>
      )}
    </div>
  );
};
