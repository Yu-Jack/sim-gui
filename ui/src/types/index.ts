export interface Version {
  id: string;
  name: string;
  createdAt: string;
  path: string;
  supportBundleName: string;
}

export interface Workspace {
  name: string;
  displayName?: string;
  createdAt: string;
  versions: Version[];
}

export interface UpdateStatus {
  updateAvailable: boolean;
  currentCommit: string;
  latestCommit: string;
  lastChecked: string;
  message: string;
}
