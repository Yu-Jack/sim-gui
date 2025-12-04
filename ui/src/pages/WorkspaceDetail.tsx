import React, { useEffect, useState, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { Upload, List, Search } from 'lucide-react';
import { getWorkspace, getSimulatorStatus } from '../api/client';
import type { Workspace } from '../types';
import { UploadArea } from '../components/workspace/UploadArea';
import { VersionList } from '../components/workspace/VersionList';
import { ResourceHistory } from '../components/workspace/ResourceHistory';

type Tab = 'upload' | 'versions' | 'search';

export const WorkspaceDetail: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [statuses, setStatuses] = useState<Record<string, { running: boolean; ready: boolean }>>({});
  const [activeTab, setActiveTab] = useState<Tab>('versions');

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

  if (!workspace || !name) return <div>Loading...</div>;

  const tabs: { id: Tab; label: string; icon: React.ElementType }[] = [
    { id: 'upload', label: 'Upload Bundle', icon: Upload },
    { id: 'versions', label: 'Versions', icon: List },
    { id: 'search', label: 'Resource Search', icon: Search },
  ];

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-gray-900">Workspace: {workspace.name}</h1>
      </div>

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
      </div>
    </div>
  );
};

