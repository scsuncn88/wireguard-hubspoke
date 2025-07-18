import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Dashboard } from './components/Dashboard';
import { Nodes } from './components/Nodes';
import { Topology } from './components/Topology';
import { Policies } from './components/Policies';
import { Settings } from './components/Settings';
import { ApiProvider } from './services/ApiContext';
import './App.css';

function App() {
  return (
    <ApiProvider>
      <div className="App">
        <Layout>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/nodes" element={<Nodes />} />
            <Route path="/topology" element={<Topology />} />
            <Route path="/policies" element={<Policies />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </Layout>
      </div>
    </ApiProvider>
  );
}

export default App;