import React from 'react';
import { Link, useLocation } from 'react-router-dom';

interface LayoutProps {
  children: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const location = useLocation();

  const navItems = [
    { path: '/', label: 'Dashboard', icon: '📊' },
    { path: '/nodes', label: 'Nodes', icon: '🖥️' },
    { path: '/topology', label: 'Topology', icon: '🌐' },
    { path: '/policies', label: 'Policies', icon: '🛡️' },
    { path: '/settings', label: 'Settings', icon: '⚙️' },
  ];

  return (
    <div className="layout">
      <nav className="sidebar">
        <div className="sidebar-header">
          <h2>WireGuard SD-WAN</h2>
        </div>
        <ul className="nav-list">
          {navItems.map((item) => (
            <li key={item.path} className={`nav-item ${location.pathname === item.path ? 'active' : ''}`}>
              <Link to={item.path} className="nav-link">
                <span className="nav-icon">{item.icon}</span>
                <span className="nav-label">{item.label}</span>
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      <main className="main-content">
        <div className="container">
          {children}
        </div>
      </main>
      <style jsx>{`
        .layout {
          display: flex;
          min-height: 100vh;
        }

        .sidebar {
          width: 250px;
          background: #2c3e50;
          color: white;
          padding: 20px 0;
          position: fixed;
          height: 100vh;
          overflow-y: auto;
        }

        .sidebar-header {
          padding: 0 20px 20px;
          border-bottom: 1px solid #34495e;
        }

        .sidebar-header h2 {
          margin: 0;
          font-size: 1.5rem;
          color: #ecf0f1;
        }

        .nav-list {
          list-style: none;
          padding: 0;
          margin: 20px 0 0;
        }

        .nav-item {
          margin: 0;
        }

        .nav-link {
          display: flex;
          align-items: center;
          padding: 15px 20px;
          color: #bdc3c7;
          text-decoration: none;
          transition: all 0.3s;
        }

        .nav-link:hover {
          background: #34495e;
          color: #ecf0f1;
        }

        .nav-item.active .nav-link {
          background: #3498db;
          color: white;
        }

        .nav-icon {
          margin-right: 10px;
          font-size: 1.2rem;
        }

        .nav-label {
          font-size: 1rem;
        }

        .main-content {
          margin-left: 250px;
          padding: 20px;
          flex: 1;
          background: #f8f9fa;
          min-height: 100vh;
        }

        @media (max-width: 768px) {
          .sidebar {
            width: 60px;
          }

          .sidebar-header h2 {
            display: none;
          }

          .nav-label {
            display: none;
          }

          .main-content {
            margin-left: 60px;
          }
        }
      `}</style>
    </div>
  );
};