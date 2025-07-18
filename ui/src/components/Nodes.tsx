import React, { useState, useEffect } from 'react';
import { nodeApi, Node } from '../services/api';
import { useApi } from '../services/ApiContext';

export const Nodes: React.FC = () => {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [showModal, setShowModal] = useState(false);
  const [editingNode, setEditingNode] = useState<Node | null>(null);
  const { setError } = useApi();

  useEffect(() => {
    loadNodes();
  }, [currentPage]);

  const loadNodes = async () => {
    try {
      setLoading(true);
      const response = await nodeApi.getNodes({ page: currentPage, per_page: 10 });
      
      if (response.success && response.data) {
        setNodes(response.data);
        setTotalPages(response.pagination.total_pages);
      }
    } catch (error) {
      setError('Failed to load nodes');
      console.error('Nodes error:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (node: Node) => {
    if (window.confirm(`Are you sure you want to delete node "${node.name}"?`)) {
      try {
        await nodeApi.deleteNode(node.id);
        loadNodes();
      } catch (error) {
        setError('Failed to delete node');
        console.error('Delete error:', error);
      }
    }
  };

  const handleEdit = (node: Node) => {
    setEditingNode(node);
    setShowModal(true);
  };

  const handleSave = async (nodeData: Partial<Node>) => {
    try {
      if (editingNode) {
        await nodeApi.updateNode(editingNode.id, nodeData);
      } else {
        await nodeApi.createNode(nodeData as Omit<Node, 'id' | 'created_at' | 'updated_at'>);
      }
      setShowModal(false);
      setEditingNode(null);
      loadNodes();
    } catch (error) {
      setError('Failed to save node');
      console.error('Save error:', error);
    }
  };

  const getStatusBadge = (status: string) => {
    const statusClass = `status-badge status-${status}`;
    return <span className={statusClass}>{status}</span>;
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString() + ' ' + new Date(dateString).toLocaleTimeString();
  };

  if (loading) {
    return <div className="loading">Loading nodes</div>;
  }

  return (
    <div className="nodes">
      <div className="page-header">
        <h1>Nodes</h1>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          Add Node
        </button>
      </div>

      <div className="card">
        <div className="card-header">
          <h3 className="card-title">Node List</h3>
          <button className="btn btn-secondary" onClick={loadNodes}>
            Refresh
          </button>
        </div>

        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Type</th>
              <th>Status</th>
              <th>IP Address</th>
              <th>Endpoint</th>
              <th>Last Updated</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {nodes.map((node) => (
              <tr key={node.id}>
                <td>{node.name}</td>
                <td>
                  <span className={`node-type ${node.node_type}`}>
                    {node.node_type}
                  </span>
                </td>
                <td>{getStatusBadge(node.status)}</td>
                <td>{node.allocated_ip}</td>
                <td>{node.endpoint || '-'}</td>
                <td>{formatDate(node.updated_at)}</td>
                <td>
                  <button
                    className="btn btn-secondary btn-sm"
                    onClick={() => handleEdit(node)}
                  >
                    Edit
                  </button>
                  <button
                    className="btn btn-danger btn-sm"
                    onClick={() => handleDelete(node)}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {totalPages > 1 && (
          <div className="pagination">
            <button
              className="btn btn-secondary"
              disabled={currentPage === 1}
              onClick={() => setCurrentPage(currentPage - 1)}
            >
              Previous
            </button>
            <span>Page {currentPage} of {totalPages}</span>
            <button
              className="btn btn-secondary"
              disabled={currentPage === totalPages}
              onClick={() => setCurrentPage(currentPage + 1)}
            >
              Next
            </button>
          </div>
        )}
      </div>

      {showModal && (
        <NodeModal
          node={editingNode}
          onSave={handleSave}
          onClose={() => {
            setShowModal(false);
            setEditingNode(null);
          }}
        />
      )}

      <style jsx>{`
        .nodes {
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

        .node-type {
          padding: 4px 8px;
          border-radius: 4px;
          font-size: 12px;
          font-weight: 500;
          text-transform: uppercase;
        }

        .node-type.hub {
          background-color: #fff3cd;
          color: #856404;
        }

        .node-type.spoke {
          background-color: #cce7ff;
          color: #004085;
        }

        .pagination {
          display: flex;
          justify-content: center;
          align-items: center;
          gap: 15px;
          margin-top: 20px;
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

interface NodeModalProps {
  node: Node | null;
  onSave: (node: Partial<Node>) => void;
  onClose: () => void;
}

const NodeModal: React.FC<NodeModalProps> = ({ node, onSave, onClose }) => {
  const [formData, setFormData] = useState({
    name: node?.name || '',
    node_type: node?.node_type || 'spoke',
    endpoint: node?.endpoint || '',
    port: node?.port || 51820,
    public_key: node?.public_key || '',
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave(formData);
  };

  return (
    <div className="modal">
      <div className="modal-content">
        <div className="modal-header">
          <h3 className="modal-title">
            {node ? 'Edit Node' : 'Add New Node'}
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
            <label className="form-label">Type</label>
            <select
              className="form-control"
              value={formData.node_type}
              onChange={(e) => setFormData({ ...formData, node_type: e.target.value as 'hub' | 'spoke' })}
            >
              <option value="spoke">Spoke</option>
              <option value="hub">Hub</option>
            </select>
          </div>

          <div className="form-group">
            <label className="form-label">Endpoint</label>
            <input
              type="text"
              className="form-control"
              value={formData.endpoint}
              onChange={(e) => setFormData({ ...formData, endpoint: e.target.value })}
              placeholder="example.com"
            />
          </div>

          <div className="form-group">
            <label className="form-label">Port</label>
            <input
              type="number"
              className="form-control"
              value={formData.port}
              onChange={(e) => setFormData({ ...formData, port: parseInt(e.target.value) || 51820 })}
              min="1"
              max="65535"
            />
          </div>

          <div className="form-group">
            <label className="form-label">Public Key</label>
            <textarea
              className="form-control"
              value={formData.public_key}
              onChange={(e) => setFormData({ ...formData, public_key: e.target.value })}
              rows={3}
              placeholder="Base64 encoded public key"
              required
            />
          </div>

          <div className="modal-actions">
            <button type="button" className="btn btn-secondary" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              {node ? 'Update' : 'Create'}
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
        `}</style>
      </div>
    </div>
  );
};