package admin

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"esp-rainmaker-server/internal/config"
	"esp-rainmaker-server/internal/model"
	"esp-rainmaker-server/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

//go:embed static/*
var staticFiles embed.FS

var adminSecret = "admin-secret-change-this"

func RegisterRoutes(r *gin.Engine) {
	adminAPI := r.Group("/admin/api")
	{
		adminAPI.POST("/login", AdminLogin)
		adminAPI.GET("/dashboard", AdminAuth(), Dashboard)
		adminAPI.GET("/users", AdminAuth(), ListUsers)
		adminAPI.POST("/users", AdminAuth(), CreateUser)
		adminAPI.DELETE("/users/:id", AdminAuth(), DeleteUser)
		adminAPI.GET("/nodes", AdminAuth(), ListNodes)
		adminAPI.DELETE("/nodes/:id", AdminAuth(), DeleteNode)
		adminAPI.GET("/automations", AdminAuth(), ListAutomations)
		adminAPI.GET("/config", AdminAuth(), GetConfig)
		adminAPI.PUT("/config", AdminAuth(), UpdateConfig)
		adminAPI.GET("/logs", AdminAuth(), GetLogs)
		adminAPI.POST("/users/:id/reset-password", AdminAuth(), ResetPassword)
	}

	// Serve embedded admin frontend
	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Serve admin panel
		if path == "/admin" || path == "/admin/" || path == "/" {
			data, err := fs.ReadFile(staticFS, "index.html")
			if err == nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", data)
				return
			}
		}
		if len(path) > 1 {
			// Try to serve static file
			filePath := path[1:] // remove leading /
			if _, err := fs.Stat(staticFS, filePath); err == nil {
				c.Request.URL.Path = "/" + filePath
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"description": "not found"})
	})
}

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
			tokenStr = tokenStr[7:]
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(adminSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// POST /admin/api/login
func AdminLogin(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if body.Username != config.AppConfig.Admin.Username || body.Password != config.AppConfig.Admin.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "admin",
		"exp": time.Now().Add(time.Duration(config.AppConfig.Admin.SessionTTL) * time.Second).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(adminSecret))

	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

// GET /admin/api/dashboard
func Dashboard(c *gin.Context) {
	userCount, _ := store.CountUsers()
	nodeCount, _ := store.CountNodes()
	onlineCount, _ := store.CountOnlineNodes()

	c.JSON(http.StatusOK, gin.H{
		"user_count":       userCount,
		"node_count":       nodeCount,
		"online_node_count": onlineCount,
		"server_version":   "1.0.0",
		"uptime":           time.Since(startTime).String(),
	})
}

var startTime = time.Now()

// GET /admin/api/users
func ListUsers(c *gin.Context) {
	users, err := store.ListUsers(100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	if users == nil {
		users = []model.User{}
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// POST /admin/api/users
func CreateUser(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	hash, err := store.HashPassword(body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Email:        body.Email,
		PasswordHash: hash,
		UserID:       uuid.New().String(),
		IsVerified:   true,
	}
	if err := store.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// DELETE /admin/api/users/:id
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := store.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// POST /admin/api/users/:id/reset-password
func ResetPassword(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		NewPassword string `json:"new_password"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	hash, _ := store.HashPassword(body.NewPassword)
	store.UpdateUserPassword(id, hash)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GET /admin/api/nodes
func ListNodes(c *gin.Context) {
	rows, err := store.DB.Query(`SELECT n.id, n.owner_id, n.node_type, n.is_online, n.last_seen, n.created_at FROM nodes n ORDER BY n.created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	var nodes []map[string]interface{}
	for rows.Next() {
		var id, ownerID, nodeType string
		var isOnline bool
		var lastSeen, createdAt time.Time
		rows.Scan(&id, &ownerID, &nodeType, &isOnline, &lastSeen, &createdAt)
		nodes = append(nodes, map[string]interface{}{
			"node_id":    id,
			"owner_id":   ownerID,
			"node_type":  nodeType,
			"is_online":  isOnline,
			"last_seen":  lastSeen,
			"created_at": createdAt,
		})
	}
	if nodes == nil {
		nodes = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

// DELETE /admin/api/nodes/:id
func DeleteNode(c *gin.Context) {
	id := c.Param("id")
	store.DeleteNode(id)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GET /admin/api/automations
func ListAutomations(c *gin.Context) {
	rows, err := store.DB.Query(`SELECT id, user_id, name, created_at FROM automations ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	var automations []map[string]interface{}
	for rows.Next() {
		var id, userID, name string
		var createdAt time.Time
		rows.Scan(&id, &userID, &name, &createdAt)
		automations = append(automations, map[string]interface{}{
			"automation_id": id,
			"user_id":       userID,
			"name":          name,
			"created_at":    createdAt,
		})
	}
	if automations == nil {
		automations = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"automations": automations})
}

// GET /admin/api/config
func GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": gin.H{
			"host": config.AppConfig.Server.Host,
			"port": config.AppConfig.Server.Port,
		},
		"jwt": gin.H{
			"access_token_ttl":  config.AppConfig.JWT.AccessTokenTTL,
			"refresh_token_ttl": config.AppConfig.JWT.RefreshTokenTTL,
		},
	})
}

// PUT /admin/api/config
func UpdateConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "restart required for changes to take effect"})
}

// GET /admin/api/logs
func GetLogs(c *gin.Context) {
	logs, err := store.GetAPILogs(100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	if logs == nil {
		logs = []model.APILogEntry{}
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
