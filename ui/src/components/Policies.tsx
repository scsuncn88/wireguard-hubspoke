import React, { useState, useEffect } from 'react';
import { policyApi, Policy } from '../services/api';
import { useApi } from '../services/ApiContext';

export const Policies: React.FC = () => {
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<Policy | null>(null);
  const { setError } = useApi();

  useEffect(() => {
    loadPolicies();
  }, []);

  const loadPolicies = async () => {
    try {
      setLoading(true);
      const response = await policyApi.getPolicies();
      
      if (response.success && response.data) {
        setPolicies(response.data);
      }
    } catch (error) {
      setError('Failed to load policies');
      console.error('Policies error:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (policy: Policy) => {
    if (window.confirm(`Are you sure you want to delete policy "${policy.name}"?`)) {
      try {
        await policyApi.deletePolicy(policy.id);
        loadPolicies();
      } catch (error) {
        setError('Failed to delete policy');
        console.error('Delete error:', error);
      }
    }
  };

  const handleEdit = (policy: Policy) => {
    setEditingPolicy(policy);
    setShowModal(true);
  };

  const handleSave = async (policyData: Partial<Policy>) => {
    try {
      if (editingPolicy) {
        await policyApi.updatePolicy(editingPolicy.id, policyData);
      } else {
        await policyApi.createPolicy(policyData as Omit<Policy, 'id' | 'created_at' | 'updated_at'>);
      }
      setShowModal(false);
      setEditingPolicy(null);
      loadPolicies();
    } catch (error) {
      setError('Failed to save policy');
      console.error('Save error:', error);
    }
  };

  const getActionBadge = (action: string) => {
    const actionClass = `action-badge action-${action}`;
    return <span className={actionClass}>{action}</span>;
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString() + ' ' + new Date(dateString).toLocaleTimeString();
  };

  if (loading) {
    return <div className="loading">Loading policies</div>;
  }

  return (
    <div className="policies">
      <div className="page-header">
        <h1>Access Control Policies</h1>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          Add Policy
        </button>
      </div>

      <div className="card">
        <div className="card-header">
          <h3 className="card-title">Policy List</h3>
          <button className="btn btn-secondary" onClick={loadPolicies}>
            Refresh
          </button>
        </div>

        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Action</th>
              <th>Source</th>
              <th>Destination</th>
              <th>Priority</th>
              <th>Status</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {policies.map((policy) => (
              <tr key={policy.id}>
                <td>{policy.name}</td>
                <td>{getActionBadge(policy.action)}</td>
                <td>{policy.source_cidr || policy.source_node_id || '-'}</td>
                <td>{policy.destination_cidr || policy.destination_node_id || '-'}</td>
                <td>{policy.priority}</td>
                <td>
                  <span className={`status-badge ${policy.enabled ? 'status-active' : 'status-inactive'}`}>
                    {policy.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </td>
                <td>{formatDate(policy.created_at)}</td>
                <td>
                  <button
                    className="btn btn-secondary btn-sm"
                    onClick={() => handleEdit(policy)}
                  >
                    Edit
                  </button>
                  <button
                    className="btn btn-danger btn-sm"
                    onClick={() => handleDelete(policy)}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {policies.length === 0 && (
          <div className="empty-state">
            <p>No policies configured yet.</p>
            <button className="btn btn-primary" onClick={() => setShowModal(true)}>
              Create First Policy
            </button>
          </div>
        )}
      </div>

      {showModal && (
        <PolicyModal
          policy={editingPolicy}
          onSave={handleSave}
          onClose={() => {
            setShowModal(false);
            setEditingPolicy(null);
          }}
        />
      )}

      <style jsx>{`
        .policies {
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

        .btn-sm {
          padding: 4px 8px;
          font-size: 12px;
          margin-right: 8px;
        }

        .action-badge {
          padding: 4px 8px;
          border-radius: 4px;
          font-size: 12px;
          font-weight: 500;
          text-transform: uppercase;
        }

        .action-badge.action-allow {
          background-color: #d4edda;
          color: #155724;
        }

        .action-badge.action-deny {
          background-color: #f8d7da;
          color: #721c24;
        }

        .empty-state {
          text-align: center;
          padding: 40px;
          color: #666;
        }

        .empty-state p {
          margin-bottom: 20px;
        }

        @media (max-width: 768px) {
          .page-header {
            flex-direction: column;
            gap: 15px;
          }

          .table {
            font-size: 14px;
          }
        }
      `}</style>
    </div>
  );
};

interface PolicyModalProps {
  policy: Policy | null;
  onSave: (policy: Partial<Policy>) => void;
  onClose: () => void;
}

const PolicyModal: React.FC<PolicyModalProps> = ({ policy, onSave, onClose }) => {
  const [formData, setFormData] = useState({
    name: policy?.name || '',
    description: policy?.description || '',
    source_cidr: policy?.source_cidr || '',
    destination_cidr: policy?.destination_cidr || '',
    protocol: policy?.protocol || 'tcp',
    port: policy?.port || '',
    action: policy?.action || 'allow',
    priority: policy?.priority || 100,
    enabled: policy?.enabled ?? true,
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const submitData = {
      ...formData,
      port: formData.port ? parseInt(formData.port as string) : undefined,
    };
    onSave(submitData);
  };

  return (
    <div className="modal">
      <div className="modal-content">
        <div className="modal-header">
          <h3 className="modal-title">
            {policy ? 'Edit Policy' : 'Add New Policy'}
          </h3>
          <button className="close" onClick={onClose}>×</button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label">Name</label>
            <input
              type="text"
              className="form-control"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
          </div>

          <div className="form-group">
            <label className="form-label">Description</label>
            <textarea
              className="form-control"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              rows={3}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Action</label>
            <select
              className="form-control"
              value={formData.action}
              onChange={(e) => setFormData({ ...formData, action: e.target.value as 'allow' | 'deny' })}
            >
              <option value="allow">Allow</option>
              <option value="deny">Deny</option>
            </select>
          </div>

          <div className="form-group">
            <label className="form-label">Source CIDR</label>
            <input
              type="text"
              className="form-control"
              value={formData.source_cidr}
              onChange={(e) => setFormData({ ...formData, source_cidr: e.target.value })}
              placeholder="10.0.0.0/8"
            />
          </div>

          <div className="form-group">
            <label className="form-label">Destination CIDR</label>
            <input
              type="text"
              className="form-control"
              value={formData.destination_cidr}
              onChange={(e) => setFormData({ ...formData, destination_cidr: e.target.value })}
              placeholder="10.0.0.0/8"
            />
          </div>

          <div className="form-group">
            <label className="form-label">Protocol</label>
            <select
              className="form-control"
              value={formData.protocol}
              onChange={(e) => setFormData({ ...formData, protocol: e.target.value })}
            >
              <option value="tcp">TCP</option>
              <option value="udp">UDP</option>
              <option value="icmp">ICMP</option>
              <option value="any">Any</option>
            </select>
          </div>

          <div className="form-group">
            <label className="form-label">Port</label>
            <input
              type="number"
              className="form-control"
              value={formData.port}
              onChange={(e) => setFormData({ ...formData, port: e.target.value })}
              min="1"
              max="65535"
              placeholder="80"
            />
          </div>

          <div className="form-group">
            <label className="form-label">Priority</label>
            <input
              type="number"
              className="form-control"
              value={formData.priority}
              onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 100 })}
              min="1"
              max="1000"
            />
          </div>

          <div className="form-group">
            <label className="form-label">
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
              />
              Enabled
            </label>
          </div>

          <div className="modal-actions">
            <button type="button" className="btn btn-secondary" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              {policy ? 'Update' : 'Create'}
            </button>
          </div>
        </form>

        <style jsx>{`
          .modal-actions {
            display: flex;
            justify-content: flex-end;
            gap: 10px;
            margin-top: 20px;
          }

          .form-group label input[type="checkbox"] {
            margin-right: 8px;
          }
        `}</style>
      </div>
    </div>
  );
};