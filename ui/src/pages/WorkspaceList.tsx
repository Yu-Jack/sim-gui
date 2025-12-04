import React, { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { AxiosError } from 'axios';
import { Plus, Folder } from 'lucide-react';
import { getWorkspaces, createWorkspace } from '../api/client';
import type { Workspace } from '../types';

export const WorkspaceList: React.FC = () => {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [isCreating, setIsCreating] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [error, setError] = useState<string | null>(null);

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

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-gray-900">Workspaces</h1>
        <button
          onClick={() => { setIsCreating(true); setError(null); }}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700"
        >
          <Plus className="w-4 h-4 mr-2" />
          New Workspace
        </button>
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

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {workspaces.map((ws) => (
          <Link
            key={ws.name}
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
                      {ws.name}
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
        ))}
      </div>
    </div>
  );
};
