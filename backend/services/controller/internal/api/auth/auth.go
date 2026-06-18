package auth

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func getJwtKey() []byte {
	jwtKey, ok := os.LookupEnv("SECRET_API_KEY")
	if !ok || jwtKey == "" {
		return []byte("supersecretkey")
	}
	return []byte(jwtKey)
}

type JWTClaim struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// TokenInfo contains the decoded JWT claims.
type TokenInfo struct {
	Email    string
	Role     string
	TenantID string
}

func GenerateJWT(email, username, role, tenantID string) (tokenString string, err error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &JWTClaim{
		Username: username,
		Email:    email,
		Role:     role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "OctoLink",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(getJwtKey())
	return
}

func ValidateToken(signedToken string) (email string, err error) {
	info, err := ValidateTokenFull(signedToken)
	if err != nil {
		return "", err
	}
	return info.Email, nil
}

func ValidateTokenFull(signedToken string) (info TokenInfo, err error) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return getJwtKey(), nil
		},
	)
	if err != nil {
		log.Println(err)
		return
	}

	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		err = errors.New("couldn't parse claims")
		return
	}

	info = TokenInfo{
		Email:    claims.Email,
		Role:     claims.Role,
		TenantID: claims.TenantID,
	}
	return
}

// CheckPermission reports whether the given role has the required permission.
// The permission map mirrors db.BuiltinRolePermissions but lives here to avoid
// an import cycle between api/auth and db.
func CheckPermission(role, permission string) bool {
	builtinPerms := map[string][]string{
		"super_admin":  {"devices:read", "devices:write", "users:read", "users:write", "tenants:manage", "roles:manage"},
		"tenant_admin": {"devices:read", "devices:write", "users:read", "users:write"},
		"operator":     {"devices:read", "devices:write"},
		"viewer":       {"devices:read"},
	}
	perms, ok := builtinPerms[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}
