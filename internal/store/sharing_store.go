package store

import (
	"esp-rainmaker-server/internal/model"
	"fmt"
	"time"
)

// ---- Sharing Requests ----

func CreateSharingRequest(req *model.SharingRequest) error {
	_, err := DB.Exec(`INSERT INTO sharing_requests (id, node_id, group_id, from_user_id, to_user_name, status) VALUES (?, ?, ?, ?, ?, ?)`,
		req.ID, req.NodeID, req.GroupID, req.FromUserID, req.ToUserName, req.Status)
	return err
}

func GetSharingRequestsForUser(userName string, isPrimary bool, startReqID string) ([]model.SharingRequest, error) {
	query := `SELECT id, node_id, group_id, from_user_id, to_user_name, status, created_at FROM sharing_requests WHERE 1=1`
	var args []interface{}

	if isPrimary {
		query += ` AND from_user_id = (SELECT user_id FROM users WHERE email = ?)`
		args = append(args, userName)
	} else {
		query += ` AND to_user_name = ? AND status = 'pending'`
		args = append(args, userName)
	}
	if startReqID != "" {
		query += ` AND id > ?`
		args = append(args, startReqID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []model.SharingRequest
	for rows.Next() {
		var r model.SharingRequest
		if err := rows.Scan(&r.ID, &r.NodeID, &r.GroupID, &r.FromUserID, &r.ToUserName, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}
	return requests, nil
}

func GetSharingRequestByID(requestID string) (*model.SharingRequest, error) {
	r := &model.SharingRequest{}
	err := DB.QueryRow(`SELECT id, node_id, group_id, from_user_id, to_user_name, status, created_at FROM sharing_requests WHERE id = ?`, requestID).
		Scan(&r.ID, &r.NodeID, &r.GroupID, &r.FromUserID, &r.ToUserName, &r.Status, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func UpdateSharingRequestStatus(requestID, status string) error {
	_, err := DB.Exec(`UPDATE sharing_requests SET status = ? WHERE id = ?`, status, requestID)
	return err
}

func DeleteSharingRequest(requestID string) error {
	_, err := DB.Exec(`DELETE FROM sharing_requests WHERE id = ?`, requestID)
	return err
}

func GetNodeSharing(nodeID string) ([]model.UserNode, error) {
	rows, err := DB.Query(`SELECT id, user_id, node_id, role, created_at FROM user_nodes WHERE node_id = ?`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.UserNode
	for rows.Next() {
		var un model.UserNode
		rows.Scan(&un.ID, &un.UserID, &un.NodeID, &un.Role, &un.CreatedAt)
		result = append(result, un)
	}
	return result, nil
}

func RemoveSharing(nodeIDs []string, userName string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	for _, nodeID := range nodeIDs {
		var userID string
		err := tx.QueryRow(`SELECT user_id FROM users WHERE email = ?`, userName).Scan(&userID)
		if err != nil {
			continue
		}
		tx.Exec(`DELETE FROM user_nodes WHERE node_id = ? AND user_id = ? AND role = 'secondary'`, nodeID, userID)
	}
	return tx.Commit()
}

// ---- Groups ----

func CreateGroup(g *model.Group) error {
	_, err := DB.Exec(`INSERT INTO groups (id, name, owner_id, fabric_details) VALUES (?, ?, ?, ?)`,
		g.ID, g.Name, g.OwnerID, g.FabricDetails)
	return err
}

func UpdateGroup(groupID, name, fabricDetails string) error {
	_, err := DB.Exec(`UPDATE groups SET name = ?, fabric_details = ? WHERE id = ?`, name, fabricDetails, groupID)
	return err
}

func DeleteGroup(groupID string) error {
	_, err := DB.Exec(`DELETE FROM groups WHERE id = ?`, groupID)
	return err
}

func GetGroupsForUser(userID string, startID, groupID string, fabricDetails, nodeList bool) ([]model.Group, error) {
	query := `SELECT id, name, owner_id, fabric_details, created_at FROM groups WHERE owner_id = ?`
	args := []interface{}{userID}
	if startID != "" {
		query += ` AND id > ?`
		args = append(args, startID)
	}
	if groupID != "" {
		query += ` AND id = ?`
		args = append(args, groupID)
	}
	query += ` ORDER BY id`

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []model.Group
	for rows.Next() {
		var g model.Group
		rows.Scan(&g.ID, &g.Name, &g.OwnerID, &g.FabricDetails, &g.CreatedAt)
		groups = append(groups, g)
	}
	return groups, nil
}

func AddNodeToGroup(groupID, nodeID string) error {
	_, err := DB.Exec(`INSERT OR IGNORE INTO group_nodes (group_id, node_id) VALUES (?, ?)`, groupID, nodeID)
	return err
}

func RemoveNodeFromGroup(groupID, nodeID string) error {
	_, err := DB.Exec(`DELETE FROM group_nodes WHERE group_id = ? AND node_id = ?`, groupID, nodeID)
	return err
}

func GetGroupNodes(groupID string) ([]string, error) {
	rows, err := DB.Query(`SELECT node_id FROM group_nodes WHERE group_id = ?`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// ---- Automations ----

func CreateAutomation(a *model.Automation) error {
	_, err := DB.Exec(`INSERT INTO automations (id, user_id, name, automation_json) VALUES (?, ?, ?, ?)`,
		a.ID, a.UserID, a.Name, a.AutomationJSON)
	return err
}

func GetAutomationsForUser(userID, startID string) ([]model.Automation, error) {
	query := `SELECT id, user_id, name, automation_json, created_at, updated_at FROM automations WHERE user_id = ?`
	args := []interface{}{userID}
	if startID != "" {
		query += ` AND id > ?`
		args = append(args, startID)
	}
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var automations []model.Automation
	for rows.Next() {
		var a model.Automation
		rows.Scan(&a.ID, &a.UserID, &a.Name, &a.AutomationJSON, &a.CreatedAt, &a.UpdatedAt)
		automations = append(automations, a)
	}
	return automations, nil
}

func GetAutomationByID(automationID string) (*model.Automation, error) {
	a := &model.Automation{}
	err := DB.QueryRow(`SELECT id, user_id, name, automation_json, created_at, updated_at FROM automations WHERE id = ?`, automationID).
		Scan(&a.ID, &a.UserID, &a.Name, &a.AutomationJSON, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func UpdateAutomation(automationID, name, automationJSON string) error {
	_, err := DB.Exec(`UPDATE automations SET name = ?, automation_json = ?, updated_at = ? WHERE id = ?`,
		name, automationJSON, time.Now(), automationID)
	return err
}

func DeleteAutomation(automationID string) error {
	_, err := DB.Exec(`DELETE FROM automations WHERE id = ?`, automationID)
	return err
}

// ---- Time Series ----

func InsertTimeSeriesData(ts *model.TimeSeriesData) error {
	_, err := DB.Exec(`INSERT INTO timeseries_data (node_id, param_name, data_type, value, timestamp) VALUES (?, ?, ?, ?, ?)`,
		ts.NodeID, ts.ParamName, ts.DataType, ts.Value, ts.Timestamp)
	return err
}

func GetTimeSeriesData(nodeID, paramName, aggregate string, startTime, endTime int64, startID string, limit int) ([]model.TimeSeriesData, string, error) {
	query := `SELECT id, node_id, param_name, data_type, value, timestamp FROM timeseries_data
		WHERE node_id = ? AND param_name = ? AND timestamp BETWEEN datetime(?, 'unixepoch') AND datetime(?, 'unixepoch')`
	args := []interface{}{nodeID, paramName, startTime, endTime}

	if startID != "" {
		query += ` AND id > ?`
		args = append(args, startID)
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, limit+1)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var results []model.TimeSeriesData
	var lastID int64
	for rows.Next() {
		var ts model.TimeSeriesData
		rows.Scan(&ts.ID, &ts.NodeID, &ts.ParamName, &ts.DataType, &ts.Value, &ts.Timestamp)
		results = append(results, ts)
		lastID = ts.ID
	}

	var nextID string
	if len(results) > limit {
		nextID = fmt.Sprintf("%d", lastID)
		results = results[:limit]
	}
	return results, nextID, nil
}

// ---- OTA ----

func CreateOTAJob(job *model.OTAJob) error {
	_, err := DB.Exec(`INSERT INTO ota_jobs (id, node_id, fw_url, fw_version, status) VALUES (?, ?, ?, ?, ?)`,
		job.ID, job.NodeID, job.FWURL, job.FWVersion, job.Status)
	return err
}

func GetOTAJob(jobID string) (*model.OTAJob, error) {
	j := &model.OTAJob{}
	err := DB.QueryRow(`SELECT id, node_id, fw_url, fw_version, status, created_at, updated_at FROM ota_jobs WHERE id = ?`, jobID).
		Scan(&j.ID, &j.NodeID, &j.FWURL, &j.FWVersion, &j.Status, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func GetOTAJobByNode(nodeID string) (*model.OTAJob, error) {
	j := &model.OTAJob{}
	err := DB.QueryRow(`SELECT id, node_id, fw_url, fw_version, status, created_at, updated_at FROM ota_jobs WHERE node_id = ? ORDER BY created_at DESC LIMIT 1`, nodeID).
		Scan(&j.ID, &j.NodeID, &j.FWURL, &j.FWVersion, &j.Status, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func UpdateOTAJobStatus(jobID, status string) error {
	_, err := DB.Exec(`UPDATE ota_jobs SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now(), jobID)
	return err
}

// ---- Device Tokens ----

func RegisterDeviceToken(dt *model.DeviceToken) error {
	_, err := DB.Exec(`INSERT OR REPLACE INTO device_tokens (id, user_id, token, platform) VALUES (?, ?, ?, ?)`,
		dt.ID, dt.UserID, dt.Token, dt.Platform)
	return err
}

func UnregisterDeviceToken(token string) error {
	_, err := DB.Exec(`DELETE FROM device_tokens WHERE token = ?`, token)
	return err
}

// ---- Command Requests ----

func CreateCommandRequest(cmd *model.CommandRequest) error {
	_, err := DB.Exec(`INSERT INTO command_requests (request_id, node_id, cmd, data, timeout, is_base64) VALUES (?, ?, ?, ?, ?, ?)`,
		cmd.RequestID, cmd.NodeID, cmd.Cmd, cmd.Data, cmd.Timeout, cmd.IsBase64)
	return err
}

func GetCommandRequest(requestID string) (*model.CommandRequest, error) {
	c := &model.CommandRequest{}
	err := DB.QueryRow(`SELECT request_id, node_id, status, response, description FROM command_requests WHERE request_id = ?`, requestID).
		Scan(&c.RequestID, &c.NodeID, &c.Status, &c.Response, &c.Description)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ---- API Logs ----

func InsertAPILog(method, path, userID, ip string, status int, durationMs int64) error {
	_, err := DB.Exec(`INSERT INTO api_logs (method, path, user_id, ip, status, duration_ms) VALUES (?, ?, ?, ?, ?, ?)`,
		method, path, userID, ip, status, durationMs)
	return err
}

func GetAPILogs(limit, offset int) ([]model.APILogEntry, error) {
	rows, err := DB.Query(`SELECT id, method, path, user_id, ip, status, duration_ms, created_at FROM api_logs ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []model.APILogEntry
	for rows.Next() {
		var l model.APILogEntry
		rows.Scan(&l.ID, &l.Method, &l.Path, &l.UserID, &l.IP, &l.Status, &l.Duration, &l.CreatedAt)
		logs = append(logs, l)
	}
	return logs, nil
}
