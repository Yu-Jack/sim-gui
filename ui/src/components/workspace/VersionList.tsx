import React, { useState, useRef, useEffect } from 'react';
import { FileArchive, Play, Square, Download, Trash2, Circle, Loader2, Eraser, ChevronDown, Copy } from 'lucide-react';
import { getKubeconfigUrl, startSimulator, stopSimulator, deleteVersion, cleanVersionImage } from '../../api/client';
import type { Workspace } from '../../types';
import { useToast } from '../../contexts/ToastContext';
import { ConfirmDialog } from '../ConfirmDialog';

interface VersionListProps {
  workspace: Workspace;
  statuses: Record<string, { running: boolean; ready: boolean }>;
  onRefresh: () => void;
}

export const VersionList: React.FC<VersionListProps> = ({
  workspace,
  statuses,
  onRefresh,
}) => {
  const [loading, setLoading] = useState<Record<string, string | null>>({}); // versionID -> action ('start', 'stop', 'delete')
  const [openCopyMenu, setOpenCopyMenu] = useState<string | null>(null); // versionID of open menu
  const copyMenuRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const { showSuccess, showError, showInfo } = useToast();
  const [confirmDialog, setConfirmDialog] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
    onConfirm: () => void;
    variant?: 'danger' | 'warning';
  }>({
    isOpen: false,
    title: '',
    message: '',
    onConfirm: () => {},
  });

  const handleStart = async (versionID: string) => {
    setLoading(prev => ({ ...prev, [versionID]: 'start' }));
    try {
      await startSimulator(workspace.name, versionID);
      onRefresh();
    } catch (error) {
      console.error('Failed to start simulator', error);
      showError('Failed to start simulator');
    } finally {
      setLoading(prev => ({ ...prev, [versionID]: null }));
    }
  };

  const handleStop = async (versionID: string) => {
    setLoading(prev => ({ ...prev, [versionID]: 'stop' }));
    try {
      await stopSimulator(workspace.name, versionID);
      onRefresh();
    } catch (error) {
      console.error('Failed to stop simulator', error);
      showError('Failed to stop simulator');
    } finally {
      setLoading(prev => ({ ...prev, [versionID]: null }));
    }
  };

  const handleDelete = async (versionID: string) => {
    setConfirmDialog({
      isOpen: true,
      title: 'Delete Version',
      message: 'Are you sure you want to delete this version? This will also remove any running containers.',
      variant: 'danger',
      onConfirm: async () => {
        setConfirmDialog({ ...confirmDialog, isOpen: false });
        setLoading(prev => ({ ...prev, [versionID]: 'delete' }));
        try {
          await deleteVersion(workspace.name, versionID);
          showSuccess('Version deleted successfully');
          onRefresh();
        } catch (error) {
          console.error('Failed to delete version', error);
          showError('Failed to delete version');
        } finally {
          setLoading(prev => ({ ...prev, [versionID]: null }));
        }
      },
    });
  };

  const handleCleanImage = async (versionID: string) => {
    setConfirmDialog({
      isOpen: true,
      title: 'Clean Docker Image',
      message: 'Are you sure you want to clean the Docker image for this version? This will free up disk space.',
      variant: 'warning',
      onConfirm: async () => {
        setConfirmDialog({ ...confirmDialog, isOpen: false });
        setLoading(prev => ({ ...prev, [versionID]: 'clean' }));
        try {
          await cleanVersionImage(workspace.name, versionID);
          showSuccess('Docker image cleaned successfully!');
          onRefresh();
        } catch (error) {
          console.error('Failed to clean image', error);
          showError('Failed to clean image');
        } finally {
          setLoading(prev => ({ ...prev, [versionID]: null }));
        }
      },
    });
  };

  const handleCopyK9sCommand = (versionID: string) => {
    const kubeconfigPath = `/tmp/sim-${workspace.name}-${versionID}.kubeconfig`;
    const cmd = `curl -s -o ${kubeconfigPath} http://localhost:8080${getKubeconfigUrl(workspace.name, versionID)} && k9s --kubeconfig ${kubeconfigPath}`;
    navigator.clipboard.writeText(cmd);
    showSuccess('Copied k9s command to clipboard!');
    setOpenCopyMenu(null);
  };

  const handleCopyExportCommand = (versionID: string) => {
    const kubeconfigPath = `/tmp/sim-${workspace.name}-${versionID}.kubeconfig`;
    const cmd = `curl -s -o ${kubeconfigPath} http://localhost:8080${getKubeconfigUrl(workspace.name, versionID)} && export KUBECONFIG=${kubeconfigPath}`;
    navigator.clipboard.writeText(cmd);
    showSuccess('Copied export command to clipboard!');
    setOpenCopyMenu(null);
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (openCopyMenu && copyMenuRefs.current[openCopyMenu] && !copyMenuRefs.current[openCopyMenu]?.contains(event.target as Node)) {
        setOpenCopyMenu(null);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [openCopyMenu]);

  return (
    <>
      <ConfirmDialog
        isOpen={confirmDialog.isOpen}
        title={confirmDialog.title}
        message={confirmDialog.message}
        onConfirm={confirmDialog.onConfirm}
        onCancel={() => setConfirmDialog({ ...confirmDialog, isOpen: false })}
        variant={confirmDialog.variant}
      />
      <div className="bg-white shadow sm:rounded-md">
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
                  <div className="relative" ref={(el) => (copyMenuRefs.current[version.id] = el)}>
                    <button
                      onClick={() => setOpenCopyMenu(openCopyMenu === version.id ? null : version.id)}
                      disabled={!isRunning || !!isLoading}
                      className={`inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md ${isRunning ? 'text-green-700 bg-green-100 hover:bg-green-200' : 'text-gray-400 bg-gray-100 cursor-not-allowed'}`}
                    >
                      <Download className="h-4 w-4 mr-1" />
                      Export Kubeconfig
                      <ChevronDown className="h-3 w-3 ml-1" />
                    </button>
                    {openCopyMenu === version.id && isRunning && (
                      <div className="absolute left-0 mt-2 w-52 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 z-50">
                        <div className="py-1" role="menu">
                          <button
                            onClick={() => handleCopyK9sCommand(version.id)}
                            className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex items-center"
                            role="menuitem"
                          >
                            <Copy className="h-4 w-4 mr-2" />
                            k9s Command
                          </button>
                          <button
                            onClick={() => handleCopyExportCommand(version.id)}
                            className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex items-center"
                            role="menuitem"
                          >
                            <Copy className="h-4 w-4 mr-2" />
                            Export Command
                          </button>
                          <a
                            href={getKubeconfigUrl(workspace.name, version.id)}
                            download="kubeconfig"
                            className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex items-center"
                            role="menuitem"
                            onClick={() => setOpenCopyMenu(null)}
                          >
                            <Download className="h-4 w-4 mr-2" />
                            Download Kubeconfig
                          </a>
                        </div>
                      </div>
                    )}
                  </div>
                  <button
                    onClick={() => handleCleanImage(version.id)}
                    className={`inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md ${!isRunning ? 'text-orange-700 bg-orange-100 hover:bg-orange-200' : 'text-gray-400 bg-gray-100 cursor-not-allowed'}`}
                    disabled={isRunning || !!isLoading}
                    title="Clean Docker image to free up disk space (only available when simulator is stopped)"
                  >
                    {isLoading === 'clean' ? <Loader2 className="h-4 w-4 mr-1 animate-spin" /> : <Eraser className="h-4 w-4 mr-1" />}
                    Clean Image
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
    </>
  );
};
