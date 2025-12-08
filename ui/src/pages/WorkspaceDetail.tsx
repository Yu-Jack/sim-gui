import React, { useEffect, useState, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { Upload, List, Search, Pencil, Folder, Trash2, Loader2 } from 'lucide-react';
import { getWorkspace, getSimulatorStatus, renameWorkspace, cleanAllWorkspaceImages } from '../api/client';
import type { Workspace } from '../types';
import { UploadArea } from '../components/workspace/UploadArea';
import { getWorkspaceDisplayName, getWorkspaceEditableName } from '../utils/workspace';
import { VersionList } from '../components/workspace/VersionList';
import { ResourceHistory } from '../components/workspace/ResourceHistory';
import NodeExplorer from '../components/workspace/NodeExplorer.tsx';

type Tab = 'upload' | 'versions' | 'search' | 'explorer';

export const WorkspaceDetail: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [statuses, setStatuses] = useState<Record<string, { running: boolean; ready: boolean }>>({});
  const [activeTab, setActiveTab] = useState<Tab>('versions');

  const [isRenaming, setIsRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState('');
  const [showRenameModal, setShowRenameModal] = useState(false);
  const [isCleaning, setIsCleaning] = useState(false);

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
    const init = async () => {
      await loadWorkspace();
    };
    init();
  }, [loadWorkspace]);

  useEffect(() => {
    if (workspace) {
      const initStatus = async () => {
        await loadStatuses();
      };
      initStatus();
      const interval = setInterval(loadStatuses, 5000); // Poll every 5 seconds
      return () => clearInterval(interval);
    }
  }, [workspace, loadStatuses]);

  const handleCleanAll = async () => {
    if (!name) return;
    if (!confirm('Are you sure you want to stop all simulators and clean all Docker images for this workspace? This will free up disk space but you\'ll need to restart simulators after.')) return;
    
    setIsCleaning(true);
    try {
      await cleanAllWorkspaceImages(name);
      alert('All containers and images cleaned successfully!');
      await loadWorkspace();
      await loadStatuses();
    } catch (error) {
      console.error('Failed to clean all images', error);
      alert('Failed to clean all images');
    } finally {
      setIsCleaning(false);
    }
  };

  const handleRenameSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!workspace || !name || !renameValue.trim() || renameValue === getWorkspaceEditableName(workspace)) return;
    
    setIsRenaming(true);
    try {
      await renameWorkspace(name, renameValue);
      setShowRenameModal(false);
      await loadWorkspace();
    } catch (err) {
        console.error(err);
        alert('Failed to rename workspace');
    } finally {
        setIsRenaming(false);
    }
  };

  if (!workspace || !name) return <div>Loading...</div>;

  const tabs: { id: Tab; label: string; icon: React.ElementType }[] = [
    { id: 'upload', label: 'Upload Bundle', icon: Upload },
    { id: 'versions', label: 'Versions', icon: List },
    { id: 'search', label: 'Resource Search', icon: Search },
    { id: 'explorer', label: 'Node Explorer', icon: Folder },
  ];

  return (
    <div className="space-y-6 h-[calc(100vh-100px)] flex flex-col">
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold text-gray-900">
              Workspace: {getWorkspaceDisplayName(workspace)}
            </h1>
            <button 
                onClick={() => {
                    setRenameValue(getWorkspaceEditableName(workspace));
                    setShowRenameModal(true);
                }}
                className="p-1 text-gray-400 hover:text-indigo-600 transition-colors"
            >
                <Pencil className="h-5 w-5" />
            </button>
        </div>
        <button
          onClick={handleCleanAll}
          disabled={isCleaning}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-orange-600 hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed"
          title="Stop all simulators and clean all Docker images"
        >
          {isCleaning ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Trash2 className="h-4 w-4 mr-2" />}
          {isCleaning ? 'Cleaning...' : 'Clean All Images'}
        </button>
      </div>

      {showRenameModal && (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Rename Workspace</h3>
            <form onSubmit={handleRenameSubmit}>
              <input
                type="text"
                value={renameValue}
                onChange={(e) => setRenameValue(e.target.value)}
                className="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2 mb-4"
                autoFocus
                disabled={isRenaming}
              />
              <div className="flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setShowRenameModal(false)}
                  disabled={isRenaming}
                  className="px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isRenaming}
                  className="px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700"
                >
                  {isRenaming ? 'Renaming...' : 'Rename'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8" aria-label="Tabs">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`
                  group inline-flex items-center py-4 px-1 border-b-2 font-medium text-sm
                  ${activeTab === tab.id
                    ? 'border-indigo-500 text-indigo-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  }
                `}
              >
                <Icon
                  className={`
                    -ml-0.5 mr-2 h-5 w-5
                    ${activeTab === tab.id ? 'text-indigo-500' : 'text-gray-400 group-hover:text-gray-500'}
                  `}
                />
                {tab.label}
              </button>
            );
          })}
        </nav>
      </div>

      <div className="mt-6">
        {activeTab === 'upload' && (
          <UploadArea workspaceName={name} onUploadComplete={() => {
            loadWorkspace();
            setActiveTab('versions');
          }} />
        )}

        {activeTab === 'versions' && (
          <VersionList
            workspace={workspace}
            statuses={statuses}
            onRefresh={() => {
              loadWorkspace();
              loadStatuses();
            }}
          />
        )}

        {activeTab === 'search' && (
          <ResourceHistory workspaceName={name} />
        )}

        {activeTab === 'explorer' && (
          <div className="h-[calc(100vh-200px)]">
            <NodeExplorer workspaceName={name} versions={workspace.versions} />
          </div>
        )}
      </div>
    </div>
  );
};

