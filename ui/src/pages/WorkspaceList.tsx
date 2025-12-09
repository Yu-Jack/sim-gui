import React, { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { AxiosError } from 'axios';
import { Plus, Folder, Pencil, Trash, Loader2, Trash2 } from 'lucide-react';
import { getWorkspaces, createWorkspace, renameWorkspace, deleteWorkspace, cleanAllImages } from '../api/client';
import type { Workspace } from '../types';
import { getWorkspaceDisplayName, getWorkspaceEditableName } from '../utils/workspace';
import { useToast } from '../contexts/ToastContext';
import { ConfirmDialog } from '../components/ConfirmDialog';

export const WorkspaceList: React.FC = () => {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [isCreating, setIsCreating] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [error, setError] = useState<string | null>(null);

  const [editingWorkspace, setEditingWorkspace] = useState<Workspace | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [isRenaming, setIsRenaming] = useState(false);
  const [deletingWorkspace, setDeletingWorkspace] = useState<string | null>(null);
  const [isCleaningAll, setIsCleaningAll] = useState(false);
  const { showSuccess, showError } = useToast();
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

  const loadWorkspaces = useCallback(async () => {
    try {
      const data = await getWorkspaces();
      console.log('Loaded workspaces:', data);
      setWorkspaces(data || []);
    } catch (error) {
      console.error('Failed to load workspaces', error);
    }
  }, []);

  useEffect(() => {
    loadWorkspaces();
  }, [loadWorkspaces]);


  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWorkspaceName.trim()) return;
    setError(null);
    setIsSubmitting(true);

    try {
      await createWorkspace(newWorkspaceName);
      setNewWorkspaceName('');
      setIsCreating(false);
      await loadWorkspaces();
    } catch (err) {
      const error = err as AxiosError;
      console.error('Failed to create workspace', error);
      if (error.response && error.response.data) {
        setError(typeof error.response.data === 'string' ? (error.response.data as string) : 'Failed to create workspace');
      } else {
        setError('Failed to create workspace');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRenameSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingWorkspace || !renameValue.trim() || renameValue === getWorkspaceEditableName(editingWorkspace)) return;
    
    setIsRenaming(true);
    try {
      await renameWorkspace(editingWorkspace.name, renameValue);
      setEditingWorkspace(null);
      setRenameValue('');
      await loadWorkspaces();
    } catch (err) {
        const error = err as AxiosError;
        console.error('Failed to rename workspace', error);
        showError('Failed to rename workspace');
    } finally {
        setIsRenaming(false);
    }
  };

  const handleDelete = async (name: string) => {
    setConfirmDialog({
      isOpen: true,
      title: 'Delete Workspace',
      message: 'Are you sure you want to delete this workspace? This action cannot be undone.',
      variant: 'danger',
      onConfirm: async () => {
        setConfirmDialog({ ...confirmDialog, isOpen: false });
        setDeletingWorkspace(name);
        try {
          await deleteWorkspace(name);
          await loadWorkspaces();
        } catch (error) {
          console.error('Failed to delete workspace', error);
          showError('Failed to delete workspace');
        } finally {
          setDeletingWorkspace(null);
        }
      },
    });
  };

  const handleCleanAll = async () => {
    setConfirmDialog({
      isOpen: true,
      title: 'Clean All Images',
      message: 'Are you sure you want to stop ALL simulators and clean ALL Docker images across all workspaces? This will free up significant disk space but you\'ll need to restart simulators after.',
      variant: 'warning',
      onConfirm: async () => {
        setConfirmDialog({ ...confirmDialog, isOpen: false });
        setIsCleaningAll(true);
        try {
          await cleanAllImages();
          showSuccess('All containers and images cleaned successfully!');
          await loadWorkspaces();
        } catch (error) {
          console.error('Failed to clean all images', error);
          showError('Failed to clean all images');
        } finally {
          setIsCleaningAll(false);
        }
      },
    });
  };

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
      <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-gray-900">Workspaces</h1>
        <div className="flex gap-3">
          <button
            onClick={handleCleanAll}
            disabled={isCleaningAll}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-orange-600 hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed"
            title="Stop all simulators and clean all Docker images"
          >
            {isCleaningAll ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Trash2 className="w-4 h-4 mr-2" />}
            {isCleaningAll ? 'Cleaning...' : 'Clean All Images'}
          </button>
          <button
            onClick={() => { setIsCreating(true); setError(null); }}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700"
          >
            <Plus className="w-4 h-4 mr-2" />
            New Workspace
          </button>
        </div>
      </div>

      {isCreating && (
        <div className="bg-white shadow sm:rounded-lg p-6">
          <form onSubmit={handleCreate} className="flex flex-col gap-4">
            <div className="flex gap-4">
              <input
                type="text"
                value={newWorkspaceName}
                onChange={(e) => setNewWorkspaceName(e.target.value)}
                placeholder="Workspace Name"
                className="flex-1 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm border p-2"
                autoFocus
                disabled={isSubmitting}
              />
              <button
                type="submit"
                disabled={isSubmitting}
                className={`inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-indigo-700 bg-indigo-100 hover:bg-indigo-200 ${isSubmitting ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                {isSubmitting ? 'Creating...' : 'Create'}
              </button>
              <button
                type="button"
                onClick={() => setIsCreating(false)}
                disabled={isSubmitting}
                className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
              >
                Cancel
              </button>
            </div>
            {error && (
              <div className="text-red-600 text-sm">
                {error}
              </div>
            )}
          </form>
        </div>
      )}

      {editingWorkspace && (
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
                  onClick={() => setEditingWorkspace(null)}
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

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {workspaces.map((ws) => (
          <div key={ws.name} className="relative group">
            <Link
              to={`/workspaces/${ws.name}`}
              className="block hover:shadow-lg transition-shadow duration-200"
            >
              <div className="bg-white overflow-hidden shadow rounded-lg">
                <div className="px-4 py-5 sm:p-6">
                  <div className="flex items-center">
                    <div className="flex-shrink-0 bg-indigo-500 rounded-md p-3">
                      <Folder className="h-6 w-6 text-white" />
                    </div>
                    <div className="ml-5 w-0 flex-1">
                      <dt className="text-sm font-medium text-gray-500 truncate">
                        {getWorkspaceDisplayName(ws)}
                      </dt>
                      <dd className="flex items-baseline">
                        <div className="text-2xl font-semibold text-gray-900">
                          {ws.versions?.length || 0} Versions
                        </div>
                      </dd>
                    </div>
                  </div>
                </div>
                <div className="bg-gray-50 px-4 py-4 sm:px-6">
                  <div className="text-sm">
                    <span className="text-gray-500">
                      Created {new Date(ws.createdAt).toLocaleDateString()}
                    </span>
                  </div>
                </div>
              </div>
            </Link>
            <div className="absolute top-2 right-2 flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
              <button
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setEditingWorkspace(ws);
                  setRenameValue(getWorkspaceEditableName(ws));
                }}
                className="p-2 text-gray-400 hover:text-indigo-600 bg-white rounded-full shadow-sm"
                title="Rename Workspace"
              >
                <Pencil className="h-4 w-4" />
              </button>
              <button
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  handleDelete(ws.name);
                }}
                disabled={deletingWorkspace === ws.name}
                className={`p-2 text-gray-400 hover:text-red-600 bg-white rounded-full shadow-sm ${deletingWorkspace === ws.name ? 'cursor-not-allowed opacity-50' : ''}`}
                title="Delete Workspace"
              >
                {deletingWorkspace === ws.name ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Trash className="h-4 w-4" />
                )}
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
    </>
  );
};
