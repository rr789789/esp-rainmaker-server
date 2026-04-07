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

// POST /v1/user/node_group
func CreateGroup(c *gin.Context) {
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	name, _ := body["group_name"].(string)
	userID := c.GetString("user_id")

	group := &model.Group{
		ID:      uuid.New().String(),
		Name:    name,
		OwnerID: userID,
	}
	if err := store.CreateGroup(group); err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed to create group")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"group_id":   group.ID,
		"group_name": group.Name,
	})
}

// PUT /v1/user/node_group
func UpdateGroup(c *gin.Context) {
	groupID := c.Query("group_id")
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	name, _ := body["group_name"].(string)
	fabricDetails := ""
	if fd, ok := body["fabric_details"]; ok {
		b, _ := json.Marshal(fd)
		fabricDetails = string(b)
	}

	if err := store.UpdateGroup(groupID, name, fabricDetails); err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed to update group")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DELETE /v1/user/node_group
func RemoveGroup(c *gin.Context) {
	groupID := c.Query("group_id")
	if err := store.DeleteGroup(groupID); err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed to delete group")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GET /v1/user/node_group
func GetUserGroups(c *gin.Context) {
	userID := c.GetString("user_id")
	startID := c.Query("start_id")
	groupID := c.Query("group_id")
	fabricDetails := c.Query("fabric_details") == "true"
	nodeList := c.Query("node_list") == "true"

	groups, err := store.GetGroupsForUser(userID, startID, groupID, fabricDetails, nodeList)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal error")
		return
	}
	if groups == nil {
		groups = []model.Group{}
	}

	result := []gin.H{}
	for _, g := range groups {
		item := gin.H{
			"group_id":   g.ID,
			"group_name": g.Name,
		}
		if nodeList {
			nodes, _ := store.GetGroupNodes(g.ID)
			item["node_list"] = nodes
		}
		if fabricDetails {
			var fd map[string]interface{}
			json.Unmarshal([]byte(g.FabricDetails), &fd)
			item["fabric_details"] = fd
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": result,
	})
}

// GET /v1/user/node_group (fabric details for group)
func GetFabricDetailsForGroup(c *gin.Context) {
	groupID := c.Query("group_id")
	nodeList := c.Query("node_list") == "true"

	groups, err := store.GetGroupsForUser(c.GetString("user_id"), "", groupID, true, nodeList)
	if err != nil || len(groups) == 0 {
		RespondWithError(c, http.StatusNotFound, "group not found")
		return
	}

	g := groups[0]
	var fd map[string]interface{}
	json.Unmarshal([]byte(g.FabricDetails), &fd)

	result := gin.H{
		"group_id":      g.ID,
		"group_name":    g.Name,
		"fabric_details": fd,
	}
	if nodeList {
		nodes, _ := store.GetGroupNodes(g.ID)
		result["node_list"] = nodes
	}
	c.JSON(http.StatusOK, result)
}

// PUT /v1/user/node_group (convert to fabric)
func ConvertGroupToFabric(c *gin.Context) {
	groupID := c.Query("group_id")
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	fabricDetailsBytes, _ := json.Marshal(body)
	store.UpdateGroup(groupID, "", string(fabricDetailsBytes))
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// ---- Automation ----

// POST /v1/user/node_automation
func AddAutomation(c *gin.Context) {
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	name, _ := body["automation_name"].(string)
	jsonBytes, _ := json.Marshal(body)

	a := &model.Automation{
		ID:             uuid.New().String(),
		UserID:         c.GetString("user_id"),
		Name:           name,
		AutomationJSON: string(jsonBytes),
	}
	if err := store.CreateAutomation(a); err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed to create automation")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"automation_id":   a.ID,
		"automation_name": a.Name,
	})
}

// GET /v1/user/node_automation
func GetAutomations(c *gin.Context) {
	userID := c.GetString("user_id")
	startID := c.Query("start_id")

	automations, err := store.GetAutomationsForUser(userID, startID)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal error")
		return
	}
	if automations == nil {
		automations = []model.Automation{}
	}

	var result []gin.H
	for _, a := range automations {
		var autoData map[string]interface{}
		json.Unmarshal([]byte(a.AutomationJSON), &autoData)
		result = append(result, gin.H{
			"automation_id":   a.ID,
			"automation_name": a.Name,
			"automation":      autoData,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"automations": result,
	})
}

// GET /v1/user/node_automation (by ID)
func GetAutomationWithId(c *gin.Context) {
	automationID := c.Query("automation_id")
	if automationID == "" {
		GetAutomations(c)
		return
	}

	a, err := store.GetAutomationByID(automationID)
	if err != nil || a == nil {
		RespondWithError(c, http.StatusNotFound, "automation not found")
		return
	}

	var autoData map[string]interface{}
	json.Unmarshal([]byte(a.AutomationJSON), &autoData)

	c.JSON(http.StatusOK, gin.H{
		"automation_id":   a.ID,
		"automation_name": a.Name,
		"automation":      autoData,
	})
}

// PUT /v1/user/node_automation
func UpdateAutomation(c *gin.Context) {
	automationID := c.Query("automation_id")
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	name, _ := body["automation_name"].(string)
	jsonBytes, _ := json.Marshal(body)

	if err := store.UpdateAutomation(automationID, name, string(jsonBytes)); err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed to update automation")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DELETE /v1/user/node_automation
func DeleteAutomation(c *gin.Context) {
	automationID := c.Query("automation_id")
	if err := store.DeleteAutomation(automationID); err != nil {
		RespondWithError(c, http.StatusInternalServerError, "failed to delete automation")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// ---- Time Series ----

// GET /v1/user/nodes/tsdata
func GetTimeSeriesData(c *gin.Context) {
	nodeID := c.Query("node_id")
	paramName := c.Query("param_name")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	startID := c.Query("start_id")

	var startTs, endTs int64
	if startTime != "" {
		startTs, _ = parseInt64(startTime)
	}
	if endTime != "" {
		endTs, _ = parseInt64(endTime)
	}

	data, nextID, err := store.GetTimeSeriesData(nodeID, paramName, "", startTs, endTs, startID, 100)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "internal error")
		return
	}
	if data == nil {
		data = []model.TimeSeriesData{}
	}

	c.JSON(http.StatusOK, gin.H{
		"time_series_data": data,
		"next_id":          nextID,
	})
}

// GET /v1/user/nodes/simple_tsdata
func GetSimpleTimeSeriesData(c *gin.Context) {
	GetTimeSeriesData(c)
}

// ---- OTA ----

// GET /v1/user/nodes/ota_update
func CheckFwUpdate(c *gin.Context) {
	nodeID := c.Query("node_id")
	job, err := store.GetOTAJobByNode(nodeID)
	if err != nil || job == nil {
		c.JSON(http.StatusOK, gin.H{"ota_available": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ota_available": true,
		"ota_details": gin.H{
			"ota_job_id": job.ID,
			"fw_url":     job.FWURL,
			"fw_version": job.FWVersion,
			"status":     job.Status,
		},
	})
}

// GET /v1/user/nodes/ota_status
func GetFwUpdateStatus(c *gin.Context) {
	otaJobID := c.Query("ota_job_id")
	job, err := store.GetOTAJob(otaJobID)
	if err != nil || job == nil {
		RespondWithError(c, http.StatusNotFound, "ota job not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ota_job_id": job.ID,
		"status":     job.Status,
	})
}

// POST /v1/user/nodes/ota_update
func PushFwUpdate(c *gin.Context) {
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	nodeID, _ := body["node_id"].(string)
	fwURL, _ := body["fw_url"].(string)
	fwVersion, _ := body["fw_version"].(string)

	job := &model.OTAJob{
		ID:        uuid.New().String(),
		NodeID:    nodeID,
		FWURL:     fwURL,
		FWVersion: fwVersion,
		Status:    "triggered",
	}
	store.CreateOTAJob(job)

	c.JSON(http.StatusOK, gin.H{
		"ota_job_id": job.ID,
		"status":     job.Status,
	})
}

// ---- Push Notifications ----

// POST /v1/user/push_notification/mobile_platform_endpoint
func RegisterDeviceToken(c *gin.Context) {
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	token, _ := body["mobile_device_token"].(string)
	platform, _ := body["platform"].(string)
	if platform == "" {
		platform = "GCM"
	}

	dt := &model.DeviceToken{
		ID:       uuid.New().String(),
		UserID:   c.GetString("user_id"),
		Token:    token,
		Platform: platform,
	}
	store.RegisterDeviceToken(dt)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DELETE /v1/user/push_notification/mobile_platform_endpoint
func UnregisterDeviceToken(c *gin.Context) {
	token := c.Query("mobile_device_token")
	store.UnregisterDeviceToken(token)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// ---- Command Response ----

// POST /v1/user/nodes/cmd
func SendCommandResponse(c *gin.Context) {
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	requestID := uuid.New().String()
	nodeID, _ := body["node_id"].(string)
	cmd, _ := body["cmd"].(string)
	data, _ := body["data"].(string)
	timeout := 30

	cmdReq := &model.CommandRequest{
		RequestID: requestID,
		NodeID:    nodeID,
		Cmd:       cmd,
		Data:      data,
		Timeout:   timeout,
		Status:    "pending",
	}
	store.CreateCommandRequest(cmdReq)

	c.JSON(http.StatusOK, gin.H{
		"request_id": requestID,
		"status":     "pending",
	})
}

// GET /v1/user/nodes/cmd
func GetCommandResponseStatus(c *gin.Context) {
	requestID := c.Query("request_id")
	cmd, err := store.GetCommandRequest(requestID)
	if err != nil || cmd == nil {
		RespondWithError(c, http.StatusNotFound, "request not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id":        cmd.RequestID,
		"status":            cmd.Status,
		"response_data":     cmd.Response,
		"status_description": cmd.Description,
	})
}

// POST /v1/user/assume_role
func AssumeRole(c *gin.Context) {
	var body map[string]interface{}
	if !bindJSONOrError(c, &body) {
		return
	}

	role, _ := body["role"].(string)
	_ = role

	c.JSON(http.StatusOK, gin.H{
		"access_key":    "self-hosted-access-key",
		"secret_key":    "self-hosted-secret-key",
		"session_token": "self-hosted-session-token",
	})
}

func parseInt64(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n, nil
}
