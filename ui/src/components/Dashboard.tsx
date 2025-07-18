import React, { useState, useEffect } from 'react';
import { nodeApi, metricsApi, healthApi, Metrics, HealthStatus } from '../services/api';
import { useApi } from '../services/ApiContext';

export const Dashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const { setError } = useApi();

  useEffect(() => {
    loadDashboardData();
  }, []);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      
      const [metricsResponse, healthResponse] = await Promise.all([
        metricsApi.getMetrics(),
        healthApi.getHealth(),
      ]);

      if (metricsResponse.success && metricsResponse.data) {
        setMetrics(metricsResponse.data);
      }

      if (healthResponse.success && healthResponse.data) {
        setHealth(healthResponse.data);
      }
    } catch (error) {
      setError('Failed to load dashboard data');
      console.error('Dashboard error:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">Loading dashboard</div>;
  }

  return (
    <div className="dashboard">
      <div className="page-header">
        <h1>Dashboard</h1>
        <button className="btn btn-primary" onClick={loadDashboardData}>
          Refresh
        </button>
      </div>

      {/* System Health */}
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">System Health</h3>
        </div>
        <div className="health-status">
          <div className={`health-indicator ${health?.status === 'healthy' ? 'healthy' : 'unhealthy'}`}>
            <span className="status-dot"></span>
            <span className="status-text">{health?.status || 'Unknown'}</span>
          </div>
          <div className="health-details">
            <p><strong>Version:</strong> {health?.version}</p>
            <p><strong>Last Updated:</strong> {health?.timestamp}</p>
          </div>
        </div>
      </div>

      {/* Statistics */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{metrics?.nodes_total || 0}</div>
          <div className="stat-label">Total Nodes</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{metrics?.nodes_active || 0}</div>
          <div className="stat-label">Active Nodes</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{metrics?.hubs_total || 0}</div>
          <div className="stat-label">Hub Nodes</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{metrics?.spokes_total || 0}</div>
          <div className="stat-label">Spoke Nodes</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{metrics?.policies_total || 0}</div>
          <div className="stat-label">Active Policies</div>
        </div>
      </div>

      {/* Services Status */}
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">Services Status</h3>
        </div>
        <div className="services-grid">
          {health?.services && Object.entries(health.services).map(([service, status]) => (
            <div key={service} className="service-item">
              <div className="service-name">{service}</div>
              <div className={`service-status ${status === 'healthy' ? 'healthy' : 'unhealthy'}`}>
                {status}
              </div>
            </div>
          ))}
        </div>
      </div>

      <style jsx>{`
        .dashboard {
          padding: 20px 0;
        }

        .page-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 30px;
        }

        .page-header h1 {
          margin: 0;
          color: #333;
        }

        .health-status {
          display: flex;
          align-items: center;
          gap: 30px;
        }

        .health-indicator {
          display: flex;
          align-items: center;
          gap: 10px;
        }

        .status-dot {
          width: 12px;
          height: 12px;
          border-radius: 50%;
          background-color: #dc3545;
        }

        .health-indicator.healthy .status-dot {
          background-color: #28a745;
        }

        .status-text {
          font-size: 1.1rem;
          font-weight: 500;
          text-transform: capitalize;
        }

        .health-details p {
          margin: 5px 0;
          color: #666;
        }

        .services-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
          gap: 15px;
        }

        .service-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 10px 15px;
          background: #f8f9fa;
          border-radius: 6px;
        }

        .service-name {
          font-weight: 500;
          text-transform: capitalize;
        }

        .service-status {
          padding: 4px 8px;
          border-radius: 4px;
          font-size: 12px;
          font-weight: 500;
          text-transform: uppercase;
        }

        .service-status.healthy {
          background-color: #d4edda;
          color: #155724;
        }

        .service-status.unhealthy {
          background-color: #f8d7da;
          color: #721c24;
        }

        @media (max-width: 768px) {
          .health-status {
            flex-direction: column;
            align-items: flex-start;
            gap: 15px;
          }

          .services-grid {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </div>
  );
};