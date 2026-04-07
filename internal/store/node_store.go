package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"esp-rainmaker-server/internal/model"
)

// ---- Node CRUD ----

func CreateNode(node *model.Node) error {
	_, err := DB.Exec(`INSERT OR REPLACE INTO nodes (id, secret_key, owner_id, node_type, config, status, metadata, fw_version, is_online, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.ID, node.SecretKey, node.OwnerID, node.NodeType, node.Config, node.Status, node.Metadata, node.FWVersion, node.IsOnline, node.LastSeen)
	return err
}

func GetNodeByID(nodeID string) (*model.Node, error) {
	n := &model.Node{}
	err := DB.QueryRow(`SELECT id, secret_key, owner_id, node_type, config, status, metadata, fw_version, is_online, last_seen, created_at
		FROM nodes WHERE id = ?`, nodeID).
		Scan(&n.ID, &n.SecretKey, &n.OwnerID, &n.NodeType, &n.Config, &n.Status, &n.Metadata, &n.FWVersion, &n.IsOnline, &n.LastSeen, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return n, err
}

func UpdateNodeConfig(nodeID, config string) error {
	_, err := DB.Exec(`UPDATE nodes SET config = ? WHERE id = ?`, config, nodeID)
	return err
}

func UpdateNodeStatus(nodeID string, isOnline bool) error {
	_, err := DB.Exec(`UPDATE nodes SET is_online = ?, last_seen = ? WHERE id = ?`, isOnline, time.Now(), nodeID)
	return err
}

func UpdateNodeMetadata(nodeID, metadata string) error {
	_, err := DB.Exec(`UPDATE nodes SET metadata = ? WHERE id = ?`, metadata, nodeID)
	return err
}

func DeleteNode(nodeID string) error {
	_, err := DB.Exec(`DELETE FROM nodes WHERE id = ?`, nodeID)
	return err
}

func CountNodes() (int, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&count)
	return count, err
}

func CountOnlineNodes() (int, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM nodes WHERE is_online = TRUE`).Scan(&count)
	return count, err
}

// ---- User-Node mapping ----

func AddUserNode(un *model.UserNode) error {
	_, err := DB.Exec(`INSERT OR IGNORE INTO user_nodes (id, user_id, node_id, role) VALUES (?, ?, ?, ?)`,
		un.ID, un.UserID, un.NodeID, un.Role)
	return err
}

func RemoveUserNode(userID, nodeID string) error {
	_, err := DB.Exec(`DELETE FROM user_nodes WHERE user_id = ? AND node_id = ?`, userID, nodeID)
	return err
}

func GetNodesForUser(userID string, startID string, limit int) ([]map[string]interface{}, string, error) {
	query := `SELECT n.id, n.node_type, n.config, n.status, n.metadata, n.fw_version, n.is_online, n.last_seen, n.created_at,
		un.role
		FROM nodes n JOIN user_nodes un ON n.id = un.node_id
		WHERE un.user_id = ?`
	args := []interface{}{userID}

	if startID != "" {
		query += ` AND n.id > ?`
		args = append(args, startID)
	}
	query += ` ORDER BY n.id LIMIT ?`
	args = append(args, limit+1)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var results []map[string]interface{}
	var lastID string
	for rows.Next() {
		var id, nodeType, config, status, metadata, fwVersion string
		var isOnline bool
		var lastSeen, createdAt time.Time
		var role string
		if err := rows.Scan(&id, &nodeType, &config, &status, &metadata, &fwVersion, &isOnline, &lastSeen, &createdAt, &role); err != nil {
			return nil, "", err
		}

		var configMap map[string]interface{}
		json.Unmarshal([]byte(config), &configMap)
		var statusMap map[string]interface{}
		json.Unmarshal([]byte(status), &statusMap)
		var metaMap map[string]interface{}
		json.Unmarshal([]byte(metadata), &metaMap)

		nodeDetail := map[string]interface{}{
			"id":         id,
			"role":       role,
			"node_type":  nodeType,
			"config":     configMap,
			"status":     statusMap,
			"metadata":   metaMap,
			"fw_version": fwVersion,
			"is_online":  isOnline,
			"last_seen":  lastSeen,
		}
		results = append(results, nodeDetail)
		lastID = id
	}

	var nextID string
	if len(results) > limit {
		nextID = lastID
		results = results[:limit]
	}
	return results, nextID, nil
}

func GetNodeForUser(userID, nodeID string) (*model.Node, string, error) {
	n := &model.Node{}
	var role string
	err := DB.QueryRow(`SELECT n.id, n.secret_key, n.owner_id, n.node_type, n.config, n.status, n.metadata, n.fw_version, n.is_online, n.last_seen, n.created_at, un.role
		FROM nodes n JOIN user_nodes un ON n.id = un.node_id
		WHERE un.user_id = ? AND n.id = ?`, userID, nodeID).
		Scan(&n.ID, &n.SecretKey, &n.OwnerID, &n.NodeType, &n.Config, &n.Status, &n.Metadata, &n.FWVersion, &n.IsOnline, &n.LastSeen, &n.CreatedAt, &role)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	return n, role, err
}

func GetUserNodeBySecretKey(nodeID, secretKey string) (*model.Node, error) {
	n := &model.Node{}
	err := DB.QueryRow(`SELECT id, secret_key, owner_id, node_type, config, status FROM nodes WHERE id = ? AND secret_key = ?`, nodeID, secretKey).
		Scan(&n.ID, &n.SecretKey, &n.OwnerID, &n.NodeType, &n.Config, &n.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return n, err
}

// ---- Node params ----

func GetNodeParams(nodeID string) (map[string]interface{}, error) {
	n, err := GetNodeByID(nodeID)
	if err != nil || n == nil {
		return nil, err
	}
	var config map[string]interface{}
	json.Unmarshal([]byte(n.Config), &config)
	return config, nil
}

func UpdateNodeParams(nodeID string, params map[string]interface{}) error {
	n, err := GetNodeByID(nodeID)
	if err != nil || n == nil {
		return fmt.Errorf("node not found")
	}
	var config map[string]interface{}
	json.Unmarshal([]byte(n.Config), &config)

	// Merge params into config
	for k, v := range params {
		config[k] = v
	}

	newConfig, _ := json.Marshal(config)
	return UpdateNodeConfig(nodeID, string(newConfig))
}

// ---- Node status ----

func GetNodesStatus(userID string, nodeIDs []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if len(nodeIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(nodeIDs))
	args := []interface{}{userID}
	for i, id := range nodeIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`SELECT n.id, n.status, n.is_online FROM nodes n
		JOIN user_nodes un ON n.id = un.node_id
		WHERE un.user_id = ? AND n.id IN (%s)`, strings.Join(placeholders, ","))

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, status string
		var isOnline bool
		rows.Scan(&id, &status, &isOnline)
		var statusMap map[string]interface{}
		json.Unmarshal([]byte(status), &statusMap)
		statusMap["connected"] = isOnline
		result[id] = statusMap
	}
	return result, nil
}

// ---- Mapping requests ----

func CreateMappingRequest(req *model.MappingRequest) error {
	_, err := DB.Exec(`INSERT INTO mapping_requests (id, user_id, node_id, operation, secret_key, status) VALUES (?, ?, ?, ?, ?, ?)`,
		req.ID, req.UserID, req.NodeID, req.Operation, req.SecretKey, req.Status)
	return err
}

func GetMappingRequest(requestID string) (*model.MappingRequest, error) {
	r := &model.MappingRequest{}
	err := DB.QueryRow(`SELECT id, user_id, node_id, operation, secret_key, status, created_at FROM mapping_requests WHERE id = ?`, requestID).
		Scan(&r.ID, &r.UserID, &r.NodeID, &r.Operation, &r.SecretKey, &r.Status, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

func UpdateMappingRequestStatus(requestID, status string) error {
	_, err := DB.Exec(`UPDATE mapping_requests SET status = ? WHERE id = ?`, status, requestID)
	return err
}
