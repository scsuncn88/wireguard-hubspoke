import React, { useState } from 'react';

export const Settings: React.FC = () => {
  const [settings, setSettings] = useState({
    wgSubnet: '10.100.0.0/16',
    wgPortStart: 51820,
    wgPortEnd: 51870,
    wgMTU: 1420,
    wgKeepalive: 25,
    logLevel: 'info',
    enableMetrics: true,
    enableAuditLog: true,
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: Implement settings save
    console.log('Saving settings:', settings);
  };

  const handleReset = () => {
    // TODO: Implement settings reset
    console.log('Resetting settings');
  };

  return (
    <div className="settings">
      <div className="page-header">
        <h1>Settings</h1>
      </div>

      <div className="settings-grid">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">WireGuard Configuration</h3>
          </div>
          
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="form-label">Subnet</label>
              <input
                type="text"
                className="form-control"
                value={settings.wgSubnet}
                onChange={(e) => setSettings({ ...settings, wgSubnet: e.target.value })}
                placeholder="10.100.0.0/16"
              />
              <small className="form-text">IP subnet for WireGuard network</small>
            </div>

            <div className="form-group">
              <label className="form-label">Port Range Start</label>
              <input
                type="number"
                className="form-control"
                value={settings.wgPortStart}
                onChange={(e) => setSettings({ ...settings, wgPortStart: parseInt(e.target.value) })}
                min="1"
                max="65535"
              />
            </div>

            <div className="form-group">
              <label className="form-label">Port Range End</label>
              <input
                type="number"
                className="form-control"
                value={settings.wgPortEnd}
                onChange={(e) => setSettings({ ...settings, wgPortEnd: parseInt(e.target.value) })}
                min="1"
                max="65535"
              />
            </div>

            <div className="form-group">
              <label className="form-label">MTU</label>
              <input
                type="number"
                className="form-control"
                value={settings.wgMTU}
                onChange={(e) => setSettings({ ...settings, wgMTU: parseInt(e.target.value) })}
                min="1280"
                max="1500"
              />
              <small className="form-text">Maximum Transmission Unit</small>
            </div>

            <div className="form-group">
              <label className="form-label">Persistent Keepalive</label>
              <input
                type="number"
                className="form-control"
                value={settings.wgKeepalive}
                onChange={(e) => setSettings({ ...settings, wgKeepalive: parseInt(e.target.value) })}
                min="0"
                max="65535"
              />
              <small className="form-text">Seconds between keepalive packets</small>
            </div>
          </form>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="card-title">System Settings</h3>
          </div>
          
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="form-label">Log Level</label>
              <select
                className="form-control"
                value={settings.logLevel}
                onChange={(e) => setSettings({ ...settings, logLevel: e.target.value })}
              >
                <option value="debug">Debug</option>
                <option value="info">Info</option>
                <option value="warn">Warning</option>
                <option value="error">Error</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">
                <input
                  type="checkbox"
                  checked={settings.enableMetrics}
                  onChange={(e) => setSettings({ ...settings, enableMetrics: e.target.checked })}
                />
                Enable Metrics Collection
              </label>
              <small className="form-text">Collect and expose Prometheus metrics</small>
            </div>

            <div className="form-group">
              <label className="form-label">
                <input
                  type="checkbox"
                  checked={settings.enableAuditLog}
                  onChange={(e) => setSettings({ ...settings, enableAuditLog: e.target.checked })}
                />
                Enable Audit Logging
              </label>
              <small className="form-text">Log all administrative actions</small>
            </div>
          </form>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="card-title">System Information</h3>
          </div>
          
          <div className="info-grid">
            <div className="info-item">
              <strong>Version:</strong> 1.0.0
            </div>
            <div className="info-item">
              <strong>Build:</strong> dev-20240115
            </div>
            <div className="info-item">
              <strong>Uptime:</strong> 2 days, 14 hours
            </div>
            <div className="info-item">
              <strong>Database:</strong> PostgreSQL 15.2
            </div>
            <div className="info-item">
              <strong>Go Version:</strong> 1.21.0
            </div>
            <div className="info-item">
              <strong>Platform:</strong> Linux x86_64
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Actions</h3>
          </div>
          
          <div className="actions-grid">
            <button type="submit" className="btn btn-primary" onClick={handleSubmit}>
              Save Settings
            </button>
            <button type="button" className="btn btn-secondary" onClick={handleReset}>
              Reset to Defaults
            </button>
            <button type="button" className="btn btn-danger">
              Restart Service
            </button>
          </div>
        </div>
      </div>

      <style jsx>{`
        .settings {
          padding: 20px 0;
        }

        .page-header {
          margin-bottom: 30px;
        }

        .page-header h1 {
          margin: 0;
          color: #333;
        }

        .settings-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
          gap: 20px;
        }

        .form-text {
          display: block;
          margin-top: 5px;
          font-size: 12px;
          color: #666;
        }

        .info-grid {
          display: grid;
          gap: 15px;
        }

        .info-item {
          padding: 10px 15px;
          background: #f8f9fa;
          border-radius: 6px;
        }

        .actions-grid {
          display: grid;
          gap: 15px;
        }

        .form-group label input[type="checkbox"] {
          margin-right: 8px;
        }

        @media (max-width: 768px) {
          .settings-grid {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </div>
  );
};