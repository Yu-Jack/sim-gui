import React from 'react';
import { Link, Outlet } from 'react-router-dom';

export const Layout: React.FC = () => {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              <Link to="/" className="flex-shrink-0 flex items-center gap-4">
                <img 
                  src="https://docs.harvesterhci.io//img/logo_horizontal.svg" 
                  alt="Harvester Logo" 
                  className="h-8"
                />
                <span className="text-gray-900 font-semibold text-lg border-l pl-4 border-gray-300">
                  Support Bundle Disagnostic
                </span>
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
