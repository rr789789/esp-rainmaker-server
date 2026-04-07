package api

import (
	"esp-rainmaker-server/internal/admin"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API v1
	v1 := r.Group("/v1")
	{
		// Public endpoints (no auth)
		v1.POST("/login", Login)
		v1.POST("/user", CreateUser)
		v1.PUT("/forgotpassword", ForgotPassword)
		v1.GET("/apiversions", GetSupportedVersions)
		v1.POST("/token", OAuthLogin)

		// Auth required endpoints
		auth := v1.Group("")
		auth.Use(AuthMiddleware())
		{
			// User management
			auth.POST("/logout", Logout)
			auth.PUT("/password", ChangePassword)
			auth.DELETE("/user", DeleteUserRequest)

			// Nodes
			auth.GET("/user/nodes", GetNodes)
			auth.PUT("/user/nodes", AddNode)
			auth.GET("/user/nodes/status", GetNodeStatus)
			auth.GET("/user/nodes/params", GetParamValue)
			auth.PUT("/user/nodes/params", UpdateParamValue)
			auth.GET("/user/nodes/mapping", GetMappingStatus)

			// Mapping
			auth.POST("/user/nodes/mapping/initiate", InitiateMapping)
			auth.POST("/user/nodes/mapping/verify", VerifyMapping)

			// Claiming
			auth.POST("/claim/initiate", InitiateClaiming)
			auth.POST("/claim/verify", VerifyClaiming)

			// Sharing
			auth.GET("/user/nodes/sharing/requests", GetSharingRequests)
			auth.PUT("/user/nodes/sharing/requests", UpdateSharingRequest)
			auth.DELETE("/user/nodes/sharing/requests", RemoveSharingRequest)
			auth.PUT("/user/nodes/sharing", ShareNodeWithUser)
			auth.GET("/user/nodes/sharing", GetNodeSharing)
			auth.DELETE("/user/nodes/sharing", RemoveSharing)

			// Group Sharing
			auth.PUT("/user/node_group/sharing", ShareGroupWithUser)
			auth.GET("/user/node_group/sharing/requests", GetGroupSharingRequests)
			auth.PUT("/user/node_group/sharing/requests", UpdateGroupSharingRequest)
			auth.DELETE("/user/node_group/sharing/requests", RemoveGroupSharingRequest)
			auth.GET("/user/node_group/sharing", GetGroupSharing)
			auth.DELETE("/user/node_group/sharing", RemoveGroupSharing)

			// Groups
			auth.POST("/user/node_group", CreateGroup)
			auth.PUT("/user/node_group", UpdateGroup)
			auth.DELETE("/user/node_group", RemoveGroup)
			auth.GET("/user/node_group", GetUserGroups)

			// Automation
			auth.POST("/user/node_automation", AddAutomation)
			auth.GET("/user/node_automation", GetAutomations)
			auth.PUT("/user/node_automation", UpdateAutomation)
			auth.DELETE("/user/node_automation", DeleteAutomation)

			// Time Series
			auth.GET("/user/nodes/tsdata", GetTimeSeriesData)
			auth.GET("/user/nodes/simple_tsdata", GetSimpleTimeSeriesData)

			// OTA
			auth.GET("/user/nodes/ota_update", CheckFwUpdate)
			auth.GET("/user/nodes/ota_status", GetFwUpdateStatus)
			auth.POST("/user/nodes/ota_update", PushFwUpdate)

			// Push Notifications
			auth.POST("/user/push_notification/mobile_platform_endpoint", RegisterDeviceToken)
			auth.DELETE("/user/push_notification/mobile_platform_endpoint", UnregisterDeviceToken)

			// Commands
			auth.POST("/user/nodes/cmd", SendCommandResponse)
			auth.GET("/user/nodes/cmd", GetCommandResponseStatus)

			// Assume Role
			auth.POST("/user/assume_role", AssumeRole)
		}

		// User pool 2 endpoints (aliases)
		v1.POST("/login2", Login)
		v1.POST("/user2", CreateUser)
		v1.PUT("/forgotpassword2", ForgotPassword)
		v1.PUT("/password2", ChangePassword)
		v1.POST("/logout2", Logout)
	}

	// Admin API
	admin.RegisterRoutes(r)

	return r
}
