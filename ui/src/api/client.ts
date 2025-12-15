import axios from 'axios';
import type { Workspace, UpdateStatus } from '../types';

const client = axios.create({
  baseURL: 'http://localhost:8080/api',
});

export const getWorkspaces = async () => {
  const response = await client.get<Workspace[]>('/workspaces');
  return response.data;
};

export const createWorkspace = async (name: string) => {
  const response = await client.post<Workspace>('/workspaces', { name });
  return response.data;
};

export const renameWorkspace = async (oldName: string, newName: string) => {
  await client.put(`/workspaces/${oldName}`, { name: newName });
};

export const deleteWorkspace = async (name: string) => {
  await client.delete(`/workspaces/${name}`);
};

export const getWorkspace = async (name: string) => {
  const response = await client.get<Workspace>(`/workspaces/${name}`);
  return response.data;
};

export const uploadVersion = async (workspaceName: string, files: File | File[]) => {
  const formData = new FormData();
  const fileList = Array.isArray(files) ? files : [files];
  
  fileList.forEach(file => {
    formData.append('file', file);
  });

  await client.post(`/workspaces/${workspaceName}/versions`, formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
};

export const startSimulator = async (workspaceName: string, versionID: string) => {
  await client.post(`/workspaces/${workspaceName}/versions/${versionID}/start`);
};

export const stopSimulator = async (workspaceName: string, versionID: string) => {
  await client.post(`/workspaces/${workspaceName}/versions/${versionID}/stop`);
};

export const getSimulatorStatus = async (workspaceName: string, versionID: string) => {
  const response = await client.get<{ running: boolean; ready: boolean }>(`/workspaces/${workspaceName}/versions/${versionID}/status`);
  return response.data;
};

export const getKubeconfigUrl = (workspaceName: string, versionID: string) => {
  return `/api/workspaces/${workspaceName}/versions/${versionID}/kubeconfig`;
};

export const getWorkspaceKubeconfigUrl = (workspaceName: string) => {
  return `/api/workspaces/${workspaceName}/kubeconfig`;
};

export const deleteVersion = async (workspaceName: string, versionID: string) => {
  await client.delete(`/workspaces/${workspaceName}/versions/${versionID}`);
};

export const cleanVersionImage = async (workspaceName: string, versionID: string) => {
  await client.post(`/workspaces/${workspaceName}/versions/${versionID}/clean-image`);
};

export const cleanAllWorkspaceImages = async (workspaceName: string) => {
  await client.post(`/workspaces/${workspaceName}/clean-all`);
};

export const cleanAllImages = async () => {
  await client.post('/clean-all');
};

export interface ResourceHistoryResult {
  versionID: string;
  content: string;
  error?: string;
  status: 'found' | 'not_found' | 'stopped' | 'error';
}

export const getResourceHistory = async (workspaceName: string, resource: string) => {
  const response = await client.post<ResourceHistoryResult[]>(`/workspaces/${workspaceName}/resource-history`, { resource });
  return response.data;
};

export const getNamespaces = async (workspaceName: string, versionID?: string) => {
  const response = await client.get<string[]>(`/workspaces/${workspaceName}/namespaces`, {
    params: { version: versionID }
  });
  return response.data;
};

export const getResourceTypes = async (workspaceName: string) => {
  const response = await client.get<string[]>(`/workspaces/${workspaceName}/resource-types`);
  return response.data;
};

export const getResources = async (workspaceName: string, namespace: string, resourceType: string, keyword: string, versionID?: string) => {
  const response = await client.get<string[]>(`/workspaces/${workspaceName}/resources`, {
    params: { namespace, resourceType, keyword, version: versionID }
  });
  return response.data;
};

export const startCodeServer = async (workspaceName: string, versionID: string) => {
  const response = await client.post<{ url: string }>(`/workspaces/${workspaceName}/versions/${versionID}/code-server`);
  return response.data;
};

export const getUpdateStatus = async () => {
  const response = await client.get<UpdateStatus>('/update-status');
  return response.data;
};

export interface NodeCompatibilityResult {
  nodeName: string;
  matches: boolean;
  missingLabels: Array<{ key: string; value: string }>;
}

export interface NodeToNodeCompatibility {
  sourceNode: string;
  targetNode: string;
  missingLabels: Array<{ key: string; value: string }>;
}

export interface LiveMigrationCheckResult {
  podName: string;
  nodeSelector?: Record<string, string>;
  nodeResults: NodeCompatibilityResult[];
  nodeToNodeCompatibilities?: NodeToNodeCompatibility[];
  error?: string;
}

export const checkLiveMigration = async (workspaceName: string, versionID: string, namespace: string, podName: string) => {
  const response = await client.post<LiveMigrationCheckResult>(`/workspaces/${workspaceName}/live-migration-check`, { 
    versionID,
    namespace, 
    podName 
  });
  return response.data;
};
