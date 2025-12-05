import React, { useState, useEffect, useCallback, useRef } from 'react';
import { startCodeServer } from '../../api/client';

interface NodeExplorerProps {
  workspaceName: string;
  versions: { id: string; name: string }[];
}

const NodeExplorer: React.FC<NodeExplorerProps> = ({ workspaceName, versions }) => {
  const [selectedVersion, setSelectedVersion] = useState<string>(versions.length > 0 ? versions[versions.length - 1].id : '');
  const [codeServerUrl, setCodeServerUrl] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const activeRequestRef = useRef<number>(0);

  useEffect(() => {
    if (versions.length > 0 && !selectedVersion) {
        setSelectedVersion(versions[versions.length - 1].id);
    }
  }, [versions]);

  const loadCodeServer = useCallback(async () => {
    if (!selectedVersion) return;
    
    const requestId = ++activeRequestRef.current;
    setLoading(true);
    setError('');
    setCodeServerUrl('');
    
    try {
      const data = await startCodeServer(workspaceName, selectedVersion);
      if (requestId === activeRequestRef.current) {
        setCodeServerUrl(data.url);
      }
    } catch (err: any) {
      if (requestId === activeRequestRef.current) {
        console.error('Failed to start code-server:', err);
        setError(err.message || 'Failed to start code-server');
      }
    } finally {
      if (requestId === activeRequestRef.current) {
        setLoading(false);
      }
    }
  }, [workspaceName, selectedVersion]);

  useEffect(() => {
    loadCodeServer();
  }, [loadCodeServer]);

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b flex items-center gap-4 bg-white">
        <select
          value={selectedVersion}
          onChange={(e) => setSelectedVersion(e.target.value)}
          className="border rounded px-2 py-1"
        >
          {versions.map((v) => (
            <option key={v.id} value={v.id}>
              {v.name} ({v.id})
            </option>
          ))}
        </select>
        <button 
            onClick={loadCodeServer}
            className="px-3 py-1 bg-blue-500 text-white rounded hover:bg-blue-600"
        >
            Reload Editor
        </button>
        {codeServerUrl && (
            <a 
                href={codeServerUrl} 
                target="_blank" 
                rel="noopener noreferrer"
                className="text-blue-500 hover:underline ml-auto"
            >
                Open in New Tab
            </a>
        )}
      </div>
      
      <div className="flex-1 relative bg-gray-100">
        {loading && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-lg text-gray-600">Starting VS Code Server...</div>
          </div>
        )}
        
        {error && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-red-500">Error: {error}</div>
          </div>
        )}

        {codeServerUrl && !loading && (
          <iframe 
            src={codeServerUrl} 
            className="w-full h-full border-0"
            title="VS Code Server"
            allow="clipboard-read; clipboard-write"
          />
        )}
      </div>
    </div>
  );
};

export default NodeExplorer;
