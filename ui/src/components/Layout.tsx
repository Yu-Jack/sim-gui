import React from 'react';
import { Link, Outlet } from 'react-router-dom';
import { LayoutDashboard } from 'lucide-react';

export const Layout: React.FC = () => {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              <Link to="/" className="flex-shrink-0 flex items-center text-indigo-600 font-bold text-xl">
                <LayoutDashboard className="w-6 h-6 mr-2" />
                K8s Diagnostic
              </Link>
            </div>
          </div>
        </div>
      </header>
      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <Outlet />
      </main>
    </div>
  );
};
