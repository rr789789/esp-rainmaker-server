package store

import (
	"database/sql"
	"fmt"
	"time"

	"esp-rainmaker-server/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ---- User CRUD ----

func CreateUser(user *model.User) error {
	_, err := DB.Exec(`INSERT INTO users (id, email, password_hash, user_id, is_oauth, is_admin, verification_code, is_verified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.UserID, user.IsOAuth, user.IsAdmin, user.VerificationCode, user.IsVerified)
	return err
}

func GetUserByEmail(email string) (*model.User, error) {
	u := &model.User{}
	err := DB.QueryRow(`SELECT id, email, password_hash, user_id, is_oauth, is_admin, verification_code, is_verified, created_at, updated_at
		FROM users WHERE email = ?`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.UserID, &u.IsOAuth, &u.IsAdmin, &u.VerificationCode, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func GetUserByUserID(userID string) (*model.User, error) {
	u := &model.User{}
	err := DB.QueryRow(`SELECT id, email, password_hash, user_id, is_oauth, is_admin, verification_code, is_verified, created_at, updated_at
		FROM users WHERE user_id = ?`, userID).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.UserID, &u.IsOAuth, &u.IsAdmin, &u.VerificationCode, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func UpdateUserPassword(userID, passwordHash string) error {
	_, err := DB.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE user_id = ?`, passwordHash, time.Now(), userID)
	return err
}

func SetUserVerificationCode(email, code string) error {
	_, err := DB.Exec(`UPDATE users SET verification_code = ? WHERE email = ?`, code, email)
	return err
}

func VerifyUserCode(email, code string) (bool, error) {
	res, err := DB.Exec(`UPDATE users SET is_verified = TRUE WHERE email = ? AND verification_code = ?`, email, code)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func DeleteUser(userID string) error {
	_, err := DB.Exec(`DELETE FROM users WHERE user_id = ?`, userID)
	return err
}

func ListUsers(limit, offset int) ([]model.User, error) {
	rows, err := DB.Query(`SELECT id, email, user_id, is_oauth, is_admin, is_verified, created_at FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.UserID, &u.IsOAuth, &u.IsAdmin, &u.IsVerified, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func CountUsers() (int, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// ---- Password helpers ----

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ---- JWT helpers ----

type TokenClaims struct {
	UserID string `json:"custom:user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateTokenPair(user *model.User, secret string, accessTTL, refreshTTL, idTTL int) (idToken, accessToken, refreshToken string, err error) {
	now := time.Now()

	// ID Token
	idClaims := TokenClaims{
		UserID: user.UserID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(idTTL) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	idToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, idClaims).SignedString([]byte(secret))
	if err != nil {
		return
	}

	// Access Token
	accessClaims := jwt.RegisteredClaims{
		Subject:   user.UserID,
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(accessTTL) * time.Second)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(secret))
	if err != nil {
		return
	}

	// Refresh Token
	refreshClaims := jwt.RegisteredClaims{
		Subject:   user.UserID,
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(refreshTTL) * time.Second)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(secret))
	return
}

func ValidateAccessToken(tokenStr, secret string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func ValidateRefreshToken(tokenStr, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Subject, nil
}

// ---- Refresh token store ----

func StoreRefreshToken(id, userID, token string, expiresAt time.Time) error {
	_, err := DB.Exec(`INSERT OR REPLACE INTO refresh_tokens (id, user_id, token, expires_at) VALUES (?, ?, ?, ?)`,
		id, userID, token, expiresAt)
	return err
}

func GetRefreshToken(token string) (string, error) {
	var userID string
	err := DB.QueryRow(`SELECT user_id FROM refresh_tokens WHERE token = ? AND expires_at > ?`, token, time.Now()).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return userID, err
}
