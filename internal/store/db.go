package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	DB.SetMaxOpenConns(1) // SQLite limitation
	return migrate()
}

func migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			user_id TEXT UNIQUE NOT NULL,
			is_oauth BOOLEAN DEFAULT FALSE,
			is_admin BOOLEAN DEFAULT FALSE,
			verification_code TEXT DEFAULT '',
			is_verified BOOLEAN DEFAULT TRUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id TEXT PRIMARY KEY,
			secret_key TEXT DEFAULT '',
			owner_id TEXT NOT NULL,
			node_type TEXT DEFAULT 'rainmaker',
			config TEXT DEFAULT '{}',
			status TEXT DEFAULT '{"connectivity":{"connected":false}}',
			metadata TEXT DEFAULT '{}',
			fw_version TEXT DEFAULT '',
			is_online BOOLEAN DEFAULT FALSE,
			last_seen DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_nodes (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			node_id TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'primary',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, node_id),
			FOREIGN KEY (user_id) REFERENCES users(user_id),
			FOREIGN KEY (node_id) REFERENCES nodes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			fabric_details TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS group_nodes (
			group_id TEXT NOT NULL,
			node_id TEXT NOT NULL,
			PRIMARY KEY (group_id, node_id),
			FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
			FOREIGN KEY (node_id) REFERENCES nodes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS sharing_requests (
			id TEXT PRIMARY KEY,
			node_id TEXT DEFAULT '',
			group_id TEXT DEFAULT '',
			from_user_id TEXT NOT NULL,
			to_user_name TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (from_user_id) REFERENCES users(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS automations (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			automation_json TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS timeseries_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id TEXT NOT NULL,
			param_name TEXT NOT NULL,
			data_type TEXT DEFAULT '',
			value TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (node_id) REFERENCES nodes(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ts_node_param ON timeseries_data(node_id, param_name)`,
		`CREATE INDEX IF NOT EXISTS idx_ts_timestamp ON timeseries_data(timestamp)`,
		`CREATE TABLE IF NOT EXISTS device_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT NOT NULL,
			platform TEXT DEFAULT 'GCM',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, token),
			FOREIGN KEY (user_id) REFERENCES users(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS ota_jobs (
			id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL,
			fw_url TEXT DEFAULT '',
			fw_version TEXT DEFAULT '',
			status TEXT DEFAULT 'triggered',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (node_id) REFERENCES nodes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS mapping_requests (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			node_id TEXT NOT NULL,
			operation TEXT NOT NULL DEFAULT 'add',
			secret_key TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS command_requests (
			request_id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL,
			cmd TEXT DEFAULT '',
			data TEXT DEFAULT '',
			timeout INTEGER DEFAULT 30,
			is_base64 BOOLEAN DEFAULT FALSE,
			status TEXT DEFAULT 'pending',
			response TEXT DEFAULT '',
			description TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS api_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			user_id TEXT DEFAULT '',
			ip TEXT DEFAULT '',
			status INTEGER DEFAULT 0,
			duration_ms INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		if _, err := DB.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %s: %w", stmt[:50], err)
		}
	}
	return nil
}
