package model

import "time"

type User struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	PasswordHash     string    `json:"-"`
	UserID           string    `json:"user_id"`
	IsOAuth          bool      `json:"is_oauth"`
	IsAdmin          bool      `json:"is_admin"`
	VerificationCode string    `json:"-"`
	IsVerified       bool      `json:"is_verified"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Node struct {
	ID         string    `json:"node_id"`
	SecretKey  string    `json:"secret_key"`
	OwnerID    string    `json:"owner_id"`
	NodeType   string    `json:"node_type"`
	Config     string    `json:"config"`
	Status     string    `json:"status"`
	Metadata   string    `json:"metadata"`
	FWVersion  string    `json:"fw_version"`
	IsOnline   bool      `json:"is_online"`
	LastSeen   time.Time `json:"last_seen"`
	CreatedAt  time.Time `json:"created_at"`
}

type UserNode struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	NodeID    string    `json:"node_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Group struct {
	ID           string    `json:"group_id"`
	Name         string    `json:"group_name"`
	OwnerID      string    `json:"owner_id"`
	FabricDetails string   `json:"fabric_details,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type GroupNode struct {
	GroupID string `json:"group_id"`
	NodeID  string `json:"node_id"`
}

type SharingRequest struct {
	ID           string    `json:"request_id"`
	NodeID       string    `json:"node_id,omitempty"`
	GroupID      string    `json:"group_id,omitempty"`
	FromUserID   string    `json:"from_user_id"`
	ToUserName   string    `json:"to_user_name"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type Automation struct {
	ID             string    `json:"automation_id"`
	UserID         string    `json:"user_id"`
	Name           string    `json:"automation_name"`
	AutomationJSON string    `json:"automation_json"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TimeSeriesData struct {
	ID        int64     `json:"id"`
	NodeID    string    `json:"node_id"`
	ParamName string    `json:"param_name"`
	DataType  string    `json:"data_type"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type DeviceToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
}

type OTAJob struct {
	ID        string    `json:"ota_job_id"`
	NodeID    string    `json:"node_id"`
	FWURL     string    `json:"fw_url"`
	FWVersion string    `json:"fw_version"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MappingRequest struct {
	ID          string    `json:"request_id"`
	UserID      string    `json:"user_id"`
	NodeID      string    `json:"node_id"`
	Operation   string    `json:"operation"`
	SecretKey   string    `json:"secret_key"`
	Status      string    `json:"request_status"`
	CreatedAt   time.Time `json:"request_timestamp"`
}

type CommandRequest struct {
	RequestID   string `json:"request_id"`
	NodeID      string `json:"node_id"`
	Cmd         string `json:"cmd"`
	Data        string `json:"data"`
	Timeout     int    `json:"timeout"`
	IsBase64    bool   `json:"is_base64"`
	Status      string `json:"status"`
	Response    string `json:"response_data,omitempty"`
	Description string `json:"status_description,omitempty"`
}

type APILogEntry struct {
	ID        int64     `json:"id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	UserID    string    `json:"user_id"`
	IP        string    `json:"ip"`
	Status    int       `json:"status"`
	Duration  int64     `json:"duration_ms"`
	CreatedAt time.Time `json:"created_at"`
}
