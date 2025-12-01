import React, { useEffect, useState, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { useDropzone } from 'react-dropzone';
import { Upload, FileArchive, Play, Square, Download, Terminal, Trash2, Circle, Loader2 } from 'lucide-react';
import { getWorkspace, uploadVersion, startSimulator, stopSimulator, getSimulatorStatus, getKubeconfigUrl, deleteVersion } from '../api/client';
import type { Workspace } from '../types';

export const WorkspaceDetail: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [statuses, setStatuses] = useState<Record<string, { running: boolean; ready: boolean }>>({});
  const [loading, setLoading] = useState<Record<string, string | null>>({}); // versionID -> action ('start', 'stop', 'delete')
  const [isUploading, setIsUploading] = useState(false);

  const loadWorkspace = useCallback(async () => {
    if (!name) return;
    try {
      const data = await getWorkspace(name);
      setWorkspace(data);
    } catch (error) {
      console.error('Failed to load workspace', error);
    }
  }, [name]);

  const loadStatuses = useCallback(async () => {
    if (!name || !workspace) return;
    const newStatuses: Record<string, { running: boolean; ready: boolean }> = {};
    for (const version of workspace.versions) {
      try {
        const status = await getSimulatorStatus(name, version.id);
        newStatuses[version.id] = status;
      } catch (error) {
        console.error(`Failed to load status for ${version.id}`, error);
      }
    }
    setStatuses(newStatuses);
  }, [name, workspace]);

  useEffect(() => {
    loadWorkspace();
  }, [loadWorkspace]);

  useEffect(() => {
    if (workspace) {
      loadStatuses();
      const interval = setInterval(loadStatuses, 5000); // Poll every 5 seconds
      return () => clearInterval(interval);
    }
  }, [workspace, loadStatuses]);

  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    if (!name || acceptedFiles.length === 0) return;

    setIsUploading(true);
    try {
      // Send all files at once to support split archives
      await uploadVersion(name, acceptedFiles);
      loadWorkspace();
    } catch (error) {
      console.error('Failed to upload file', error);
    } finally {
      setIsUploading(false);
    }
  }, [name, loadWorkspace]);

  const handleStart = async (versionID: string) => {
    if (!name) return;
    setLoading(prev => ({ ...prev, [versionID]: 'start' }));
    try {
      await startSimulator(name, versionID);
      loadStatuses();
    } catch (error) {
      console.error('Failed to start simulator', error);
      alert('Failed to start simulator');
    } finally {
      setLoading(prev => ({ ...prev, [versionID]: null }));
    }
  };

  const handleStop = async (versionID: string) => {
    if (!name) return;
    setLoading(prev => ({ ...prev, [versionID]: 'stop' }));
    try {
      await stopSimulator(name, versionID);
      loadStatuses();
    } catch (error) {
      console.error('Failed to stop simulator', error);
      alert('Failed to stop simulator');
    } finally {
      setLoading(prev => ({ ...prev, [versionID]: null }));
    }
  };

  const handleDelete = async (versionID: string) => {
    if (!name || !confirm('Are you sure you want to delete this version? This will also remove any running containers.')) return;
    setLoading(prev => ({ ...prev, [versionID]: 'delete' }));
    try {
      await deleteVersion(name, versionID);
      loadWorkspace();
    } catch (error) {
      console.error('Failed to delete version', error);
      alert('Failed to delete version');
    } finally {
      setLoading(prev => ({ ...prev, [versionID]: null }));
    }
  };

  const { getRootProps, getInputProps, isDragActive } = useDropzone({ onDrop });

  if (!workspace) return <div>Loading...</div>;

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-gray-900">Workspace: {workspace.name}</h1>
      </div>

      {/* Upload Area */}
      <div
        {...getRootProps()}
        className={`border-2 border-dashed rounded-lg p-12 text-center cursor-pointer transition-colors
          ${isDragActive ? 'border-indigo-500 bg-indigo-50' : 'border-gray-300 hover:border-indigo-400'}
          ${isUploading ? 'opacity-50 cursor-not-allowed' : ''}`}
      >
        <input {...getInputProps()} disabled={isUploading} />
        {isUploading ? (
          <Loader2 className="mx-auto h-12 w-12 text-indigo-500 animate-spin" />
        ) : (
          <Upload className="mx-auto h-12 w-12 text-gray-400" />
        )}
        <p className="mt-2 text-sm text-gray-600">
          {isUploading ? 'Uploading...' : 'Drag & drop support bundle here, or click to select files'}
        </p>
      </div>

      {/* Versions List */}
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <ul className="divide-y divide-gray-200">
          {workspace.versions.map((version) => {
            const status = statuses[version.id] || { running: false, ready: false };
            const isRunning = status.running;
            const isReady = status.ready;
            const isLoading = loading[version.id];

            return (
              <li key={version.id}>
                <div className="px-4 py-4 sm:px-6">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <FileArchive className="h-5 w-5 text-gray-400 mr-3" />
                      <p className="text-sm font-medium text-indigo-600 truncate">{version.id}</p>
                      {isRunning && (
                        <span className={`ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${isReady ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
                          <Circle className={`w-2 h-2 mr-1 fill-current ${isReady ? '' : 'animate-pulse'}`} />
                          {isReady ? 'Ready' : 'Initializing...'}
                        </span>
                      )}
                    </div>
                    <div className="ml-2 flex-shrink-0 flex">
                      <p className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                        {version.name}
                      </p>
                    </div>
                  </div>
                  <div className="mt-2 sm:flex sm:justify-between">
                    <div className="sm:flex">
                      <p className="flex items-center text-sm text-gray-500">
                        {version.supportBundleName}
                      </p>
                    </div>
                    <div className="mt-2 flex items-center text-sm text-gray-500 sm:mt-0 gap-2">
                      <p>
                        Uploaded {new Date(version.createdAt).toLocaleDateString()}
                      </p>
                      <button
                        onClick={() => handleDelete(version.id)}
                        className="text-red-600 hover:text-red-900 p-1 disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Delete Version"
                        disabled={!!isLoading}
                      >
                        {isLoading === 'delete' ? <Loader2 className="h-5 w-5 animate-spin" /> : <Trash2 className="h-5 w-5" />}
                      </button>
                    </div>
                  </div>
                  <div className="mt-4 flex items-center space-x-4">
                    {isRunning ? (
                      <button
                        onClick={() => handleStop(version.id)}
                        disabled={!!isLoading}
                        className="inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md text-white bg-red-600 hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {isLoading === 'stop' ? <Loader2 className="h-4 w-4 mr-1 animate-spin" /> : <Square className="h-4 w-4 mr-1 fill-current" />}
                        Stop Simulator
                      </button>
                    ) : (
                      <button
                        onClick={() => handleStart(version.id)}
                        disabled={!!isLoading}
                        className="inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md text-indigo-700 bg-indigo-100 hover:bg-indigo-200 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {isLoading === 'start' ? <Loader2 className="h-4 w-4 mr-1 animate-spin" /> : <Play className="h-4 w-4 mr-1" />}
                        Start Simulator
                      </button>
                    )}
                    <a
                      href={getKubeconfigUrl(name!, version.id)}
                      className={`inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md ${isRunning ? 'text-green-700 bg-green-100 hover:bg-green-200' : 'text-gray-400 bg-gray-100 cursor-not-allowed'}`}
                      download={isRunning ? "kubeconfig" : undefined}
                      onClick={(e) => (!isRunning || !!isLoading) && e.preventDefault()}
                    >
                      <Download className="h-4 w-4 mr-1" />
                      Kubeconfig
                    </a>
                    <button
                      onClick={() => {
                          if (!isRunning) return;
                          const cmd = `k9s --kubeconfig <(curl -s http://localhost:8080${getKubeconfigUrl(name!, version.id)})`;
                          navigator.clipboard.writeText(cmd);
                          alert('Copied k9s command to clipboard!');
                      }}
                      className={`inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md ${isRunning ? 'text-gray-700 bg-gray-100 hover:bg-gray-200' : 'text-gray-400 bg-gray-100 cursor-not-allowed'}`}
                      disabled={!isRunning || !!isLoading}
                    >
                      <Terminal className="h-4 w-4 mr-1" />
                      Copy k9s Command
                    </button>
                  </div>
                </div>
              </li>
            );
          })}
          {workspace.versions.length === 0 && (
            <li className="px-4 py-4 sm:px-6 text-center text-gray-500">
              No versions uploaded yet.
            </li>
          )}
        </ul>
      </div>
    </div>
  );
};

