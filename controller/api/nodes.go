package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/common/types"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"github.com/wg-hubspoke/wg-hubspoke/controller/services"
)

type NodesHandler struct {
	nodeService *services.NodeService
}

func NewNodesHandler(nodeService *services.NodeService) *NodesHandler {
	return &NodesHandler{
		nodeService: nodeService,
	}
}

// RegisterNode godoc
// @Summary Register a new node
// @Description Register a new hub or spoke node in the network
// @Tags nodes
// @Accept json
// @Produce json
// @Param node body types.NodeRegistrationRequest true "Node registration data"
// @Success 201 {object} types.APIResponse{data=models.Node}
// @Failure 400 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /nodes [post]
func (h *NodesHandler) RegisterNode(c *gin.Context) {
	var req types.NodeRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	node, err := h.nodeService.RegisterNode(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, types.APIResponse{
		Success: true,
		Data:    node,
		Message: "Node registered successfully",
	})
}

// GetNodes godoc
// @Summary List all nodes
// @Description Get a paginated list of all nodes
// @Tags nodes
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Param node_type query string false "Filter by node type" Enums(hub,spoke)
// @Param status query string false "Filter by status" Enums(pending,active,inactive,disabled)
// @Success 200 {object} types.PaginatedResponse{data=[]models.Node}
// @Failure 500 {object} types.APIResponse
// @Router /nodes [get]
func (h *NodesHandler) GetNodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	nodeType := c.Query("node_type")
	status := c.Query("status")

	nodes, total, err := h.nodeService.GetNodes(c.Request.Context(), page, perPage, nodeType, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))

	c.JSON(http.StatusOK, types.PaginatedResponse{
		APIResponse: types.APIResponse{
			Success: true,
			Data:    nodes,
		},
		Pagination: types.PaginationInfo{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetNode godoc
// @Summary Get a specific node
// @Description Get details of a specific node by ID
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path string true "Node ID"
// @Success 200 {object} types.APIResponse{data=models.Node}
// @Failure 404 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /nodes/{id} [get]
func (h *NodesHandler) GetNode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid node ID format",
		})
		return
	}

	node, err := h.nodeService.GetNode(c.Request.Context(), id)
	if err != nil {
		if err == services.ErrNodeNotFound {
			c.JSON(http.StatusNotFound, types.APIResponse{
				Success: false,
				Error:   "Node not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    node,
	})
}

// UpdateNode godoc
// @Summary Update a node
// @Description Update node details
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path string true "Node ID"
// @Param node body types.NodeUpdateRequest true "Node update data"
// @Success 200 {object} types.APIResponse{data=models.Node}
// @Failure 400 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /nodes/{id} [put]
func (h *NodesHandler) UpdateNode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid node ID format",
		})
		return
	}

	var req types.NodeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	node, err := h.nodeService.UpdateNode(c.Request.Context(), id, req)
	if err != nil {
		if err == services.ErrNodeNotFound {
			c.JSON(http.StatusNotFound, types.APIResponse{
				Success: false,
				Error:   "Node not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    node,
		Message: "Node updated successfully",
	})
}

// DeleteNode godoc
// @Summary Delete a node
// @Description Delete a node from the network
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path string true "Node ID"
// @Success 200 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /nodes/{id} [delete]
func (h *NodesHandler) DeleteNode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid node ID format",
		})
		return
	}

	err = h.nodeService.DeleteNode(c.Request.Context(), id)
	if err != nil {
		if err == services.ErrNodeNotFound {
			c.JSON(http.StatusNotFound, types.APIResponse{
				Success: false,
				Error:   "Node not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Node deleted successfully",
	})
}

// GetNodeConfig godoc
// @Summary Get node configuration
// @Description Get WireGuard configuration for a specific node
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path string true "Node ID"
// @Success 200 {object} types.APIResponse{data=types.NodeConfigResponse}
// @Failure 404 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Router /nodes/{id}/config [get]
func (h *NodesHandler) GetNodeConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error:   "Invalid node ID format",
		})
		return
	}

	config, err := h.nodeService.GetNodeConfig(c.Request.Context(), id)
	if err != nil {
		if err == services.ErrNodeNotFound {
			c.JSON(http.StatusNotFound, types.APIResponse{
				Success: false,
				Error:   "Node not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    config,
	})
}