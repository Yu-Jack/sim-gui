export interface Version {
  id: string;
  name: string;
  createdAt: string;
  path: string;
  supportBundleName: string;
}

export interface Workspace {
  name: string;
  createdAt: string;
  versions: Version[];
}
