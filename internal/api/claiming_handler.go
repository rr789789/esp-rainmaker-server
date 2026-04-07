package api

import (
	"net/http"
	"strings"

	"esp-rainmaker-server/internal/model"
	"esp-rainmaker-server/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// POST /v1/claim/initiate
func InitiateClaiming(c *gin.Context) {
	body, ok := bindJSONOrError(c)
	if !ok {
		return
	}

	nodeID, _ := body["node_id"].(string)
	requestID := uuid.New().String()

	c.JSON(http.StatusOK, gin.H{
		"request_id":     requestID,
		"node_id":        nodeID,
		"request_status": "initiated",
	})
}

// POST /v1/claim/verify
func VerifyClaiming(c *gin.Context) {
	body, ok := bindJSONOrError(c)
	if !ok {
		return
	}

	nodeID, _ := body["node_id"].(string)
	secretKey, _ := body["secret_key"].(string)

	// Self-hosted: auto-verify and create node
	node := &model.Node{
		ID:        nodeID,
		SecretKey: secretKey,
		OwnerID:   "",
		NodeType:  "rainmaker",
		Config:    `{"devices":[],"services":[]}`,
		Status:    `{"connectivity":{"connected":false}}`,
		Metadata:  `{"name":"Device ` + nodeID[:8] + `"}`,
	}
	store.CreateNode(node)

	c.JSON(http.StatusOK, gin.H{
		"node_id":         nodeID,
		"claim_status":    "verified",
		"secret_key":      secretKey,
	})
}

// ---- Sharing ----

// GET /v1/user/nodes/sharing/requests
func GetSharingRequests(c *gin.Context) {
	userID := c.GetString("user_id")
	isPrimary := c.Query("primary_user") == "true"
	startReqID := c.Query("start_request_id")
	startUserName := c.Query("start_user_name")

	_ = startUserName

	requests, err := store.GetSharingRequestsForUser(userID, isPrimary, startReqID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal error")
		return
	}
	if requests == nil {
		requests = []model.SharingRequest{}
	}

	c.JSON(http.StatusOK, gin.H{
		"sharing_requests": requests,
	})
}

// PUT /v1/user/nodes/sharing/requests (accept/decline)
func UpdateSharingRequest(c *gin.Context) {
	body, ok := bindJSONOrError(c)
	if !ok {
		return
	}

	requestID, _ := body["request_id"].(string)
	action, _ := body["action"].(string) // "accept" or "decline"

	req, err := store.GetSharingRequestByID(requestID)
	if err != nil || req == nil {
		RespondWithError(c, http.StatusNotFound, "request not found")
		return
	}

	if action == "accept" {
		store.UpdateSharingRequestStatus(requestID, "accepted")
		// Add sharing
		if req.NodeID != "" {
			store.AddUserNode(&model.UserNode{
				ID:     uuid.New().String(),
				UserID: req.ToUserName, // In self-hosted, to_user_name is email
				NodeID: req.NodeID,
				Role:   "secondary",
			})
		}
	} else {
		store.UpdateSharingRequestStatus(requestID, "declined")
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id":     requestID,
		"request_status": action,
	})
}

// DELETE /v1/user/nodes/sharing/requests
func RemoveSharingRequest(c *gin.Context) {
	requestID := c.Query("request_id")
	if err := store.DeleteSharingRequest(requestID); err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed to delete")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// PUT /v1/user/nodes/sharing
func ShareNodeWithUser(c *gin.Context) {
	body, ok := bindJSONOrError(c)
	if !ok {
		return
	}

	nodeID, _ := body["node_id"].(string)
	userName, _ := body["user_name"].(string) // email of the user to share with

	req := &model.SharingRequest{
		ID:         uuid.New().String(),
		NodeID:     nodeID,
		FromUserID: c.GetString("user_id"),
		ToUserName: userName,
		Status:     "pending",
	}
	store.CreateSharingRequest(req)

	c.JSON(http.StatusOK, gin.H{
		"request_id": req.ID,
		"status":     "pending",
	})
}

// GET /v1/user/nodes/sharing
func GetNodeSharing(c *gin.Context) {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		c.JSON(http.StatusOK, gin.H{"node_sharing": []interface{}{}})
		return
	}

	sharings, err := store.GetNodeSharing(nodeID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal error")
		return
	}
	if sharings == nil {
		sharings = []model.UserNode{}
	}
	c.JSON(http.StatusOK, gin.H{"node_sharing": sharings})
}

// DELETE /v1/user/nodes/sharing
func RemoveSharing(c *gin.Context) {
	nodes := c.Query("nodes")
	userName := c.Query("user_name")

	nodeIDs := []string{}
	for _, n := range splitCSV(nodes) {
		if n != "" {
			nodeIDs = append(nodeIDs, n)
		}
	}

	if err := store.RemoveSharing(nodeIDs, userName); err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed to remove sharing")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// ---- Group Sharing ----

func ShareGroupWithUser(c *gin.Context) {
	body, ok := bindJSONOrError(c)
	if !ok {
		return
	}
	groupID, _ := body["group_id"].(string)
	userName, _ := body["user_name"].(string)

	req := &model.SharingRequest{
		ID:         uuid.New().String(),
		GroupID:    groupID,
		FromUserID: c.GetString("user_id"),
		ToUserName: userName,
		Status:     "pending",
	}
	store.CreateSharingRequest(req)
	c.JSON(http.StatusOK, gin.H{"request_id": req.ID, "status": "pending"})
}

func GetGroupSharingRequests(c *gin.Context) {
	userID := c.GetString("user_id")
	isPrimary := c.Query("primary_user") == "true"
	startReqID := c.Query("start_request_id")

	requests, err := store.GetSharingRequestsForUser(userID, isPrimary, startReqID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal error")
		return
	}
	if requests == nil {
		requests = []model.SharingRequest{}
	}
	c.JSON(http.StatusOK, gin.H{"sharing_requests": requests})
}

func UpdateGroupSharingRequest(c *gin.Context) {
	UpdateSharingRequest(c) // Same logic
}

func RemoveGroupSharingRequest(c *gin.Context) {
	RemoveSharingRequest(c) // Same logic
}

func GetGroupSharing(c *gin.Context) {
	groupID := c.Query("group_id")
	_ = groupID
	c.JSON(http.StatusOK, gin.H{"group_sharing": []interface{}{}})
}

func RemoveGroupSharing(c *gin.Context) {
	groups := c.Query("groups")
	userName := c.Query("user_name")
	_ = groups
	_ = userName
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, v := range parts {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}
