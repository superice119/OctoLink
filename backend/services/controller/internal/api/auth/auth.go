package auth

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// insecureDefaultKey is the historical hardcoded signing key. It is now
// rejected outright: because the source is public, anyone could forge a
// super_admin token with it (see WS-38). The controller must be configured
// with a strong, secret SECRET_API_KEY instead.
const insecureDefaultKey = "supersecretkey"

// MinSecretLen is the minimum acceptable length for SECRET_API_KEY.
const MinSecretLen = 16

// RequireSecret validates the JWT signing secret at startup and fails closed.
// Call it once before serving traffic so the controller refuses to run with a
// missing, default, or weak key rather than silently accepting forged tokens.
func RequireSecret() error {
	key, ok := os.LookupEnv("SECRET_API_KEY")
	if !ok || key == "" {
		return errors.New("SECRET_API_KEY must be set (no insecure default key is permitted)")
	}
	if key == insecureDefaultKey {
		return fmt.Errorf("SECRET_API_KEY must not be the known default %q", insecureDefaultKey)
	}
	if len(key) < MinSecretLen {
		return fmt.Errorf("SECRET_API_KEY is too short (%d bytes); use at least %d random bytes", len(key), MinSecretLen)
	}
	return nil
}

func getJwtKey() []byte {
	jwtKey, ok := os.LookupEnv("SECRET_API_KEY")
	if !ok || jwtKey == "" || jwtKey == insecureDefaultKey {
		// Fail closed: never sign or validate tokens with a missing or
		// guessable key. RequireSecret() should have already aborted startup;
		// reaching here means a misconfiguration slipped through.
		log.Fatal("SECRET_API_KEY is unset or set to the insecure default; refusing to use a guessable JWT key")
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
