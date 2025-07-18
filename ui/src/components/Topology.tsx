import React, { useState, useEffect, useRef } from 'react';
import * as d3 from 'd3';
import { topologyApi, Topology as TopologyData } from '../services/api';
import { useApi } from '../services/ApiContext';

export const Topology: React.FC = () => {
  const [topology, setTopology] = useState<TopologyData | null>(null);
  const [loading, setLoading] = useState(true);
  const svgRef = useRef<SVGSVGElement>(null);
  const { setError } = useApi();

  useEffect(() => {
    loadTopology();
  }, []);

  useEffect(() => {
    if (topology && svgRef.current) {
      renderTopology();
    }
  }, [topology]);

  const loadTopology = async () => {
    try {
      setLoading(true);
      const response = await topologyApi.getTopology();
      
      if (response.success && response.data) {
        setTopology(response.data);
      }
    } catch (error) {
      setError('Failed to load topology');
      console.error('Topology error:', error);
    } finally {
      setLoading(false);
    }
  };

  const renderTopology = () => {
    if (!topology || !svgRef.current) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove(); // Clear previous render

    const width = 800;
    const height = 500;
    
    svg.attr('width', width).attr('height', height);

    // Create simulation
    const simulation = d3.forceSimulation(topology.nodes)
      .force('link', d3.forceLink(topology.links).id((d: any) => d.id).distance(150))
      .force('charge', d3.forceManyBody().strength(-300))
      .force('center', d3.forceCenter(width / 2, height / 2));

    // Create links
    const link = svg.append('g')
      .selectAll('line')
      .data(topology.links)
      .join('line')
      .attr('class', 'link-line')
      .attr('stroke', '#999')
      .attr('stroke-width', 2);

    // Create nodes
    const node = svg.append('g')
      .selectAll('circle')
      .data(topology.nodes)
      .join('circle')
      .attr('class', 'node-circle')
      .attr('r', 20)
      .attr('fill', (d: any) => d.node_type === 'hub' ? '#dc3545' : '#28a745')
      .call(d3.drag()
        .on('start', dragstarted)
        .on('drag', dragged)
        .on('end', dragended));

    // Create labels
    const label = svg.append('g')
      .selectAll('text')
      .data(topology.nodes)
      .join('text')
      .attr('class', 'node-text')
      .attr('text-anchor', 'middle')
      .attr('dy', 4)
      .attr('fill', 'white')
      .attr('font-size', '12px')
      .text((d: any) => d.name.substring(0, 8));

    // Create node info on hover
    node.append('title')
      .text((d: any) => `${d.name}\nType: ${d.node_type}\nStatus: ${d.status}\nIP: ${d.allocated_ip}`);

    // Update positions on simulation tick
    simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y);

      node
        .attr('cx', (d: any) => d.x)
        .attr('cy', (d: any) => d.y);

      label
        .attr('x', (d: any) => d.x)
        .attr('y', (d: any) => d.y);
    });

    function dragstarted(event: any, d: any) {
      if (!event.active) simulation.alphaTarget(0.3).restart();
      d.fx = d.x;
      d.fy = d.y;
    }

    function dragged(event: any, d: any) {
      d.fx = event.x;
      d.fy = event.y;
    }

    function dragended(event: any, d: any) {
      if (!event.active) simulation.alphaTarget(0);
      d.fx = null;
      d.fy = null;
    }
  };

  if (loading) {
    return <div className="loading">Loading topology</div>;
  }

  return (
    <div className="topology">
      <div className="page-header">
        <h1>Network Topology</h1>
        <button className="btn btn-primary" onClick={loadTopology}>
          Refresh
        </button>
      </div>

      <div className="card">
        <div className="card-header">
          <h3 className="card-title">Network Visualization</h3>
        </div>
        
        <div className="topology-container">
          <svg ref={svgRef} className="topology-svg"></svg>
        </div>

        <div className="topology-legend">
          <div className="legend-item">
            <div className="legend-color hub"></div>
            <span>Hub Node</span>
          </div>
          <div className="legend-item">
            <div className="legend-color spoke"></div>
            <span>Spoke Node</span>
          </div>
        </div>
      </div>

      {topology && (
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Network Summary</h3>
          </div>
          <div className="summary-stats">
            <div className="stat-item">
              <strong>Total Nodes:</strong> {topology.nodes.length}
            </div>
            <div className="stat-item">
              <strong>Hub Nodes:</strong> {topology.nodes.filter(n => n.node_type === 'hub').length}
            </div>
            <div className="stat-item">
              <strong>Spoke Nodes:</strong> {topology.nodes.filter(n => n.node_type === 'spoke').length}
            </div>
            <div className="stat-item">
              <strong>Active Connections:</strong> {topology.links.length}
            </div>
            <div className="stat-item">
              <strong>Online Nodes:</strong> {topology.nodes.filter(n => n.is_online).length}
            </div>
          </div>
        </div>
      )}

      <style jsx>{`
        .topology {
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

        .topology-container {
          position: relative;
          width: 100%;
          height: 500px;
          border: 1px solid #ddd;
          border-radius: 8px;
          overflow: hidden;
          background: #f8f9fa;
        }

        .topology-svg {
          width: 100%;
          height: 100%;
        }

        .topology-legend {
          display: flex;
          justify-content: center;
          gap: 30px;
          margin-top: 15px;
        }

        .legend-item {
          display: flex;
          align-items: center;
          gap: 8px;
        }

        .legend-color {
          width: 20px;
          height: 20px;
          border-radius: 50%;
          border: 2px solid #fff;
        }

        .legend-color.hub {
          background-color: #dc3545;
        }

        .legend-color.spoke {
          background-color: #28a745;
        }

        .summary-stats {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 15px;
        }

        .stat-item {
          padding: 10px 15px;
          background: #f8f9fa;
          border-radius: 6px;
        }

        @media (max-width: 768px) {
          .topology-container {
            height: 400px;
          }

          .topology-legend {
            flex-direction: column;
            align-items: center;
            gap: 10px;
          }

          .summary-stats {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </div>
  );
};