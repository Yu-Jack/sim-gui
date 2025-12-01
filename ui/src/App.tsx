import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { WorkspaceList } from './pages/WorkspaceList';
import { WorkspaceDetail } from './pages/WorkspaceDetail';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<WorkspaceList />} />
          <Route path="workspaces/:name" element={<WorkspaceDetail />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;

