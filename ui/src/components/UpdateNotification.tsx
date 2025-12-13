import React, { useEffect, useState, useCallback } from 'react';
import { getUpdateStatus } from '../api/client';
import type { UpdateStatus } from '../types';

export const UpdateNotification: React.FC = () => {
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus | null>(null);
  const [dismissed, setDismissed] = useState(false);

  const checkForUpdates = useCallback(async () => {
    try {
      const status = await getUpdateStatus();
      setUpdateStatus(status);
      // Reset dismissed state when new update is available
      if (status.updateAvailable) {
        setDismissed(false);
      }
    } catch (error) {
      console.error('Failed to check for updates:', error);
    }
  }, []);

  useEffect(() => {
    // Check for updates immediately, then every hour
    const checkImmediately = async () => {
      await checkForUpdates();
    };
    checkImmediately();

    const interval = setInterval(checkForUpdates, 15 * 60 * 1000);

    return () => clearInterval(interval);
  }, [checkForUpdates]);

  if (!updateStatus || !updateStatus.updateAvailable || dismissed) {
    return null;
  }

  return (
    <div className="bg-blue-50 border-l-4 border-blue-400 p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center">
          <div className="flex-shrink-0">
            <svg
              className="h-5 w-5 text-blue-400"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                clipRule="evenodd"
              />
            </svg>
          </div>
          <div className="ml-3">
            <p className="text-sm text-blue-700">
              {updateStatus.message}
              <span className="ml-2 text-xs text-blue-600">
                (Current: {updateStatus.currentCommit?.slice(0, 7)}, Latest: {updateStatus.latestCommit?.slice(0, 7)})
              </span>
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <button
            onClick={() => setDismissed(true)}
            className="text-blue-500 hover:text-blue-600 text-sm font-medium"
          >
            Dismiss
          </button>
        </div>
      </div>
    </div>
  );
};
