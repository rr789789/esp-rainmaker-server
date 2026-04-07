package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"esp-rainmaker-server/internal/config"
	"esp-rainmaker-server/internal/model"
	"esp-rainmaker-server/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// POST /v1/login
func Login(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "invalid request body"})
		return
	}

	// Token refresh: body contains "refreshtoken"
	if refreshToken, ok := body["refreshtoken"].(string); ok && refreshToken != "" {
		handleRefreshToken(c, refreshToken)
		return
	}

	userName, _ := body["user_name"].(string)
	password, _ := body["password"].(string)

	if userName == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"description": "user_name and password required"})
		return
	}

	user, err := store.GetUserByEmail(userName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"description": "internal error"})
		return
	}
	if user == nil || !store.CheckPassword(password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"description": "invalid credentials"})
		return
	}
	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"description": "user not verified"})
		return
	}

	idToken, accessToken, refreshToken, err := store.GenerateTokenPair(
		user, config.AppConfig.JWT.Secret,
		config.AppConfig.JWT.AccessTokenTTL,
		config.AppConfig.JWT.RefreshTokenTTL,
		config.AppConfig.JWT.IDTokenTTL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"description": "token generation failed"})
		return
	}

	// Store refresh token
	rtID := uuid.New().String()
	store.StoreRefreshToken(rtID, user.UserID, refreshToken, time.Now().Add(time.Duration(config.AppConfig.JWT.RefreshTokenTTL)*time.Second))

	c.JSON(http.StatusOK, gin.H{
		"idtoken":      idToken,
		"accesstoken":  accessToken,
		"refreshtoken": refreshToken,
	})
}

func handleRefreshToken(c *gin.Context, refreshToken string) {
	userID, err := store.ValidateRefreshToken(refreshToken, config.AppConfig.JWT.Secret)
	if err != nil || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"description": "invalid refresh token"})
		return
	}

	user, err := store.GetUserByUserID(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"description": "user not found"})
		return
	}

	idToken, accessToken, newRefreshToken, err := store.GenerateTokenPair(
		user, config.AppConfig.JWT.Secret,
		config.AppConfig.JWT.AccessTokenTTL,
		config.AppConfig.JWT.RefreshTokenTTL,
		config.AppConfig.JWT.IDTokenTTL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"description": "token generation failed"})
		return
	}

	rtID := uuid.New().String()
	store.StoreRefreshToken(rtID, user.UserID, newRefreshToken, time.Now().Add(time.Duration(config.AppConfig.JWT.RefreshTokenTTL)*time.Second))

	c.JSON(http.StatusOK, gin.H{
		"idtoken":      idToken,
		"accesstoken":  accessToken,
		"refreshtoken": newRefreshToken,
	})
}

// POST /v1/logout
func Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// POST /v1/user
func CreateUser(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "invalid request body"})
		return
	}

	userName, _ := body["user_name"].(string)
	password, _ := body["password"].(string)

	if userName == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"description": "user_name and password required"})
		return
	}

	existing, _ := store.GetUserByEmail(userName)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"description": "user already exists"})
		return
	}

	hash, err := store.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"description": "password hashing failed"})
		return
	}

	code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	user := &model.User{
		ID:               uuid.New().String(),
		Email:            userName,
		PasswordHash:     hash,
		UserID:           uuid.New().String(),
		VerificationCode: code,
		IsVerified:       true, // Auto-verify for self-hosted
	}

	if err := store.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"description": "user creation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_name":         userName,
		"verification_code": code,
	})
}

// POST /v1/user (confirm)
func ConfirmUser(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "invalid request body"})
		return
	}

	userName, _ := body["user_name"].(string)
	code, _ := body["verification_code"].(string)

	ok, err := store.VerifyUserCode(userName, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"description": "verification failed"})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"description": "invalid verification code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// PUT /v1/forgotpassword
func ForgotPassword(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "invalid request body"})
		return
	}

	userName, _ := body["user_name"].(string)
	user, _ := store.GetUserByEmail(userName)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"status": "success"}) // Don't reveal if user exists
		return
	}

	code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	store.SetUserVerificationCode(userName, code)

	c.JSON(http.StatusOK, gin.H{
		"user_name":         userName,
		"verification_code": code, // Self-hosted: return code directly
	})
}

// PUT /v1/password
func ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "invalid request body"})
		return
	}

	oldPassword, _ := body["old_password"].(string)
	newPassword, _ := body["new_password"].(string)

	user, err := store.GetUserByUserID(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"description": "user not found"})
		return
	}

	if !store.CheckPassword(oldPassword, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"description": "incorrect password"})
		return
	}

	hash, err := store.HashPassword(newPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"description": "password hashing failed"})
		return
	}

	store.UpdateUserPassword(userID, hash)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DELETE /v1/user
func DeleteUserRequest(c *gin.Context) {
	userID := c.GetString("user_id")
	request := c.Query("request")
	if request == "true" {
		code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
		user, _ := store.GetUserByUserID(userID)
		if user != nil {
			store.SetUserVerificationCode(user.Email, code)
		}
		c.JSON(http.StatusOK, gin.H{"verification_code": code})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"description": "request parameter required"})
}

// DELETE /v1/user (confirm with code)
func DeleteUserConfirm(c *gin.Context) {
	userID := c.GetString("user_id")
	code := c.Query("verification_code")
	user, _ := store.GetUserByUserID(userID)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"description": "user not found"})
		return
	}
	ok, _ := store.VerifyUserCode(user.Email, code)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"description": "invalid verification code"})
		return
	}
	store.DeleteUser(userID)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GET /v1/apiversions
func GetSupportedVersions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"supported_versions": []string{"v1"},
	})
}

// OAuth login endpoint (form-urlencoded)
// POST /v1/token
func OAuthLogin(c *gin.Context) {
	grantType := c.PostForm("grant_type")
	code := c.PostForm("code")

	if grantType == "" || code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"description": "grant_type and code required"})
		return
	}

	// For self-hosted: create/find user by code, generate tokens
	// In production, this would exchange code with the OAuth provider
	email := fmt.Sprintf("oauth_%s@selfhosted.local", code[:8])
	user, _ := store.GetUserByEmail(email)
	if user == nil {
		hash, _ := store.HashPassword(uuid.New().String())
		user = &store.User{
			ID:           uuid.New().String(),
			Email:        email,
			PasswordHash: hash,
			UserID:       uuid.New().String(),
			IsOAuth:      true,
			IsVerified:   true,
		}
		store.CreateUser(user)
	}

	idToken, accessToken, refreshToken, _ := store.GenerateTokenPair(
		user, config.AppConfig.JWT.Secret,
		config.AppConfig.JWT.AccessTokenTTL,
		config.AppConfig.JWT.RefreshTokenTTL,
		config.AppConfig.JWT.IDTokenTTL,
	)

	c.JSON(http.StatusOK, gin.H{
		"id_token":      idToken,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// ---- Auth Middleware ----

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"description": "authorization required"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"description": "invalid authorization header"})
			return
		}

		claims, err := store.ValidateAccessToken(tokenStr, config.AppConfig.JWT.Secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"description": "invalid or expired token"})
			return
		}

		c.Set("user_id", claims.Subject)
		c.Next()
	}
}

// ---- Helper ----

func bindJSONOrError(c *gin.Context) (map[string]interface{}, bool) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"description": "invalid request body"})
		return nil, false
	}
	return body, true
}

func RespondWithJSON(c *gin.Context, code int, data interface{}) {
	c.JSON(code, data)
}

func RespondWithError(c *gin.Context, code int, desc string) {
	c.JSON(code, gin.H{"description": desc})
}

// Helper for JSON param extraction
func jsonBytes(data interface{}) []byte {
	b, _ := json.Marshal(data)
	return b
}
