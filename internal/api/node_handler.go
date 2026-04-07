package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"esp-rainmaker-server/internal/model"
	"esp-rainmaker-server/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /v1/user/nodes
func GetNodes(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeDetails := c.Query("node_details")
	nodeID := c.Query("node_id")
	startID := c.Query("start_id")

	if nodeID != "" {
		// Get single node
		node, role, err := store.GetNodeForUser(userID, nodeID)
		if err != nil {
			RespondWithError(c, http.StatusInternalServerError, "internal error")
			return
		}
		if node == nil {
			RespondWithError(c, http.StatusNotFound, "node not found")
			return
		}

		var configMap, statusMap, metaMap map[string]interface{}
		json.Unmarshal([]byte(node.Config), &configMap)
		json.Unmarshal([]byte(node.Status), &statusMap)
		json.Unmarshal([]byte(node.Metadata), &metaMap)

		if nodeDetails == "true" {
			c.JSON(http.StatusOK, gin.H{
				"node_details": map[string]interface{}{
					node.ID: map[string]interface{}{
						"info": map[string]interface{}{
							"name":        metaMap["name"],
							"fw_version":  node.FWVersion,
							"type":        node.NodeType,
						},
						"config":     configMap,
						"status":     statusMap,
						"metadata":   metaMap,
						"role":       role,
					},
				},
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"node_id":    node.ID,
				"role":       role,
				"node_type":  node.NodeType,
			})
		}
		return
	}

	// List nodes
	nodes, nextID, err := store.GetNodesForUser(userID, startID, 50)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal error")
		return
	}

	if nodeDetails == "true" {
		nodeDetailsMap := gin.H{}
		for _, n := range nodes {
			nodeID := n["id"].(string)
			detail := gin.H{
				"info": gin.H{
					"name":       n["metadata"].(map[string]interface{})["name"],
					"fw_version": n["fw_version"],
					"type":       n["node_type"],
				},
				"config":   n["config"],
				"status":   n["status"],
				"metadata": n["metadata"],
				"role":     n["role"],
			}
			nodeDetailsMap[nodeID] = detail
		}
		result := gin.H{"node_details": nodeDetailsMap}
		if nextID != "" {
			result["next_id"] = nextID
		}
		c.JSON(http.StatusOK, result)
	} else {
		var nodeIDs []string
		for _, n := range nodes {
			nodeIDs = append(nodeIDs, n["id"].(string))
		}
		c.JSON(http.StatusOK, gin.H{
			"node_list": nodeIDs,
		})
	}
}

// PUT /v1/user/nodes (add/remove node)
func AddNode(c *gin.Context) {
	userID := c.GetString("user_id")

	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	operation, _ := body["operation"].(string)
	nodeID, _ := body["node_id"].(string)
	secretKey, _ := body["secret_key"].(string)

	if operation == "add" {
		// Create or update node
		node := &model.Node{
			ID:        nodeID,
			SecretKey: secretKey,
			OwnerID:   userID,
			NodeType:  "rainmaker",
			Config:    `{"devices":[],"services":[]}`,
			Status:    `{"connectivity":{"connected":false}}`,
			Metadata:  `{"name":"New Device"}`,
		}
		if err := store.CreateNode(node); err != nil {
			RespondWithError(c, http.StatusInternalServerError, "failed to add node")
			return
		}
		un := &model.UserNode{
			ID:     uuid.New().String(),
			UserID: userID,
			NodeID: nodeID,
			Role:   "primary",
		}
		store.AddUserNode(un)

		c.JSON(http.StatusOK, gin.H{
			"node_id":    nodeID,
			"secret_key": secretKey,
		})

	} else if operation == "remove" {
		store.RemoveUserNode(userID, nodeID)
		c.JSON(http.StatusOK, gin.H{"status": "success"})

	} else {
		// Update metadata
		metadata, _ := body["metadata"].(string)
		if metadata == "" {
			metadataBytes, _ := json.Marshal(body)
			metadata = string(metadataBytes)
		}
		store.UpdateNodeMetadata(nodeID, metadata)
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}

// GET /v1/user/nodes/status
func GetNodeStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeID := c.Query("node_id")

	if nodeID != "" {
		node, _, err := store.GetNodeForUser(userID, nodeID)
		if err != nil || node == nil {
			RespondWithError(c, http.StatusNotFound, "node not found")
			return
		}
		var statusMap map[string]interface{}
		json.Unmarshal([]byte(node.Status), &statusMap)
		statusMap["connected"] = node.IsOnline
		c.JSON(http.StatusOK, gin.H{
			nodeID: statusMap,
		})
		return
	}

	// All nodes for user
	statuses, err := store.GetNodesStatus(userID, nil)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal error")
		return
	}
	c.JSON(http.StatusOK, statuses)
}

// GET /v1/user/nodes/params
func GetParamValue(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeID := c.Query("node_id")

	if nodeID == "" {
		RespondWithError(c, http.StatusBadRequest, "node_id required")
		return
	}

	node, _, err := store.GetNodeForUser(userID, nodeID)
	if err != nil || node == nil {
		RespondWithError(c, http.StatusNotFound, "node not found")
		return
	}

	var configMap map[string]interface{}
	json.Unmarshal([]byte(node.Config), &configMap)
	c.JSON(http.StatusOK, configMap)
}

// PUT /v1/user/nodes/params
func UpdateParamValue(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeID := c.Query("node_id")

	if nodeID == "" {
		// Multi-node update
		var body []map[string]interface{}
		if err := c.BindJSON(&body); err != nil {
			RespondWithError(c, http.StatusBadRequest, "invalid request body")
			return
		}
		requestID := uuid.New().String()
		for _, item := range body {
			nid, _ := item["node_id"].(string)
			payload, _ := item["payload"].(map[string]interface{})
			if nid != "" && payload != nil {
				store.UpdateNodeParams(nid, payload)
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"request_id":     requestID,
			"request_status": "confirmed",
		})
		return
	}

	// Single node update
	node, _, err := store.GetNodeForUser(userID, nodeID)
	if err != nil || node == nil {
		RespondWithError(c, http.StatusNotFound, "node not found")
		return
	}

	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}
	store.UpdateNodeParams(nodeID, body)

	requestID := uuid.New().String()
	c.JSON(http.StatusOK, gin.H{
		"request_id":     requestID,
		"request_status": "confirmed",
	})
}

// GET /v1/user/nodes/mapping (poll mapping status)
func GetMappingStatus(c *gin.Context) {
	requestID := c.Query("request_id")
	userReq := c.Query("user_request")

	if requestID == "" {
		RespondWithError(c, http.StatusBadRequest, "request_id required")
		return
	}

	req, err := store.GetMappingRequest(requestID)
	if err != nil || req == nil {
		RespondWithError(c, http.StatusNotFound, "request not found")
		return
	}

	if strings.EqualFold(userReq, "true") {
		c.JSON(http.StatusOK, gin.H{
			"request_id":        req.ID,
			"request_status":    req.Status,
			"request_timestamp": req.CreatedAt.Unix(),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"request_id":     req.ID,
			"request_status": req.Status,
		})
	}
}

// POST /v1/user/nodes/mapping/initiate
func InitiateMapping(c *gin.Context) {
	userID := c.GetString("user_id")
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	nodeID, _ := body["node_id"].(string)
	secretKey, _ := body["secret_key"].(string)
	operation, _ := body["operation"].(string)
	if operation == "" {
		operation = "add"
	}

	req := &model.MappingRequest{
		ID:        uuid.New().String(),
		UserID:    userID,
		NodeID:    nodeID,
		Operation: operation,
		SecretKey: secretKey,
		Status:    "confirmed",
	}
	store.CreateMappingRequest(req)

	if operation == "add" {
		// Auto-confirm for self-hosted
		store.AddUserNode(&model.UserNode{
			ID:     uuid.New().String(),
			UserID: userID,
			NodeID: nodeID,
			Role:   "primary",
		})
		// Create node if not exists
		if n, _ := store.GetNodeByID(nodeID); n == nil {
			store.CreateNode(&model.Node{
				ID:        nodeID,
				SecretKey: secretKey,
				OwnerID:   userID,
				NodeType:  "rainmaker",
				Config:    `{"devices":[],"services":[]}`,
				Status:    `{"connectivity":{"connected":false}}`,
				Metadata:  `{"name":"Device ` + nodeID[:8] + `"}`,
			})
		}
	} else {
		store.RemoveUserNode(userID, nodeID)
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id":     req.ID,
		"request_status": "confirmed",
	})
}

// POST /v1/user/nodes/mapping/verify
func VerifyMapping(c *gin.Context) {
	userID := c.GetString("user_id")
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	requestID, _ := body["request_id"].(string)
	challengeResponse, _ := body["challenge_response"].(string)

	_ = challengeResponse // Self-hosted: auto-verify

	req, _ := store.GetMappingRequest(requestID)
	if req == nil {
		RespondWithError(c, http.StatusNotFound, "request not found")
		return
	}

	store.UpdateMappingRequestStatus(requestID, "confirmed")

	c.JSON(http.StatusOK, gin.H{
		"request_id":     requestID,
		"request_status": "confirmed",
	})
}
