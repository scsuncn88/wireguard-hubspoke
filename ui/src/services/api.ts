import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1';

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Types
export interface Node {
  id: string;
  name: string;
  node_type: 'hub' | 'spoke';
  public_key: string;
  allocated_ip: string;
  endpoint?: string;
  port?: number;
  allowed_ips?: string[];
  last_handshake?: string;
  status: 'pending' | 'active' | 'inactive' | 'disabled';
  created_at: string;
  updated_at: string;
}

export interface Policy {
  id: string;
  name: string;
  description?: string;
  source_node_id?: string;
  destination_node_id?: string;
  source_cidr?: string;
  destination_cidr?: string;
  protocol?: string;
  port?: number;
  action: 'allow' | 'deny';
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface TopologyNode {
  id: string;
  name: string;
  node_type: 'hub' | 'spoke';
  status: string;
  allocated_ip: string;
  endpoint?: string;
  is_online: boolean;
}

export interface TopologyLink {
  source: string;
  target: string;
  type: string;
}

export interface Topology {
  nodes: TopologyNode[];
  links: TopologyLink[];
}

export interface HealthStatus {
  status: string;
  version: string;
  timestamp: string;
  services: Record<string, string>;
}

export interface Metrics {
  nodes_total: number;
  nodes_active: number;
  hubs_total: number;
  spokes_total: number;
  policies_total: number;
  traffic_stats: Record<string, any>;
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

export interface PaginatedResponse<T> extends ApiResponse<T> {
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}

// API functions
export const nodeApi = {
  getNodes: async (params: { page?: number; per_page?: number; node_type?: string; status?: string } = {}) => {
    const response = await apiClient.get<PaginatedResponse<Node[]>>('/nodes', { params });
    return response.data;
  },

  getNode: async (id: string) => {
    const response = await apiClient.get<ApiResponse<Node>>(`/nodes/${id}`);
    return response.data;
  },

  createNode: async (node: Omit<Node, 'id' | 'created_at' | 'updated_at'>) => {
    const response = await apiClient.post<ApiResponse<Node>>('/nodes', node);
    return response.data;
  },

  updateNode: async (id: string, updates: Partial<Node>) => {
    const response = await apiClient.put<ApiResponse<Node>>(`/nodes/${id}`, updates);
    return response.data;
  },

  deleteNode: async (id: string) => {
    const response = await apiClient.delete<ApiResponse<void>>(`/nodes/${id}`);
    return response.data;
  },

  getNodeConfig: async (id: string) => {
    const response = await apiClient.get<ApiResponse<any>>(`/nodes/${id}/config`);
    return response.data;
  },
};

export const policyApi = {
  getPolicies: async (params: { page?: number; per_page?: number } = {}) => {
    const response = await apiClient.get<PaginatedResponse<Policy[]>>('/policies', { params });
    return response.data;
  },

  getPolicy: async (id: string) => {
    const response = await apiClient.get<ApiResponse<Policy>>(`/policies/${id}`);
    return response.data;
  },

  createPolicy: async (policy: Omit<Policy, 'id' | 'created_at' | 'updated_at'>) => {
    const response = await apiClient.post<ApiResponse<Policy>>('/policies', policy);
    return response.data;
  },

  updatePolicy: async (id: string, updates: Partial<Policy>) => {
    const response = await apiClient.put<ApiResponse<Policy>>(`/policies/${id}`, updates);
    return response.data;
  },

  deletePolicy: async (id: string) => {
    const response = await apiClient.delete<ApiResponse<void>>(`/policies/${id}`);
    return response.data;
  },
};

export const topologyApi = {
  getTopology: async () => {
    const response = await apiClient.get<ApiResponse<Topology>>('/topology');
    return response.data;
  },
};

export const healthApi = {
  getHealth: async () => {
    const response = await apiClient.get<ApiResponse<HealthStatus>>('/health');
    return response.data;
  },
};

export const metricsApi = {
  getMetrics: async () => {
    const response = await apiClient.get<ApiResponse<Metrics>>('/metrics');
    return response.data;
  },
};

// Request interceptor for authentication
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('authToken');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('authToken');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);