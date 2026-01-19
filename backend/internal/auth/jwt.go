package auth

import (
	"errors"
	"fmt"
	"time"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/utils"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	jwtLogger       = utils.NewLogger("JWT")
)

// Claims represents JWT claims with security best practices
// Uses RegisteredClaims.Subject for user UUID (standard "sub" claim)
type Claims struct {
	jwt.RegisteredClaims
}

// PendingRegistrationClaims represents JWT claims for pending registration
// Contains the pending ID that references server-side stored AccountNumber
type PendingRegistrationClaims struct {
	jwt.RegisteredClaims
	PendingID string `json:"pending_id"`
	IsPending bool   `json:"is_pending"`
}

// validateSecret ensures JWT secret meets security requirements
func validateSecret(secret string) error {
	if secret == "" {
		return errors.New("JWT_SECRET not configured")
	}
	minLen := config.GetJWTSecretMinLength()
	if minLen <= 0 {
		return errors.New("JWT_SECRET_MIN_LEN not configured")
	}
	if len(secret) < minLen {
		return fmt.Errorf("JWT_SECRET too short (minimum %d characters)", minLen)
	}
	return nil
}

func getJWTBaseConfig() (string, string, string, error) {
	secret := config.GetJWTSecret()
	if err := validateSecret(secret); err != nil {
		return "", "", "", err
	}
	issuer := config.GetJWTIssuer()
	if issuer == "" {
		return "", "", "", errors.New("JWT_ISSUER not configured")
	}
	audience := config.GetJWTAudience()
	if audience == "" {
		return "", "", "", errors.New("JWT_AUDIENCE not configured")
	}
	return secret, issuer, audience, nil
}

// SignToken creates a JWT token for the given user UUID
// Security: Uses HS256, includes iss/aud/iat/exp/nbf claims
func SignToken(userUUID string) (string, error) {
	jwtLogger.Debug("SignToken: starting token generation for userUUID: %s", userUUID)

	secret, issuer, audience, err := getJWTBaseConfig()
	if err != nil {
		jwtLogger.Error("SignToken: %v", err)
		return "", err
	}

	expiration := config.GetJWTExpiration()
	if expiration <= 0 {
		jwtLogger.Error("SignToken: invalid JWT_EXPIRATION_MINUTES")
		return "", errors.New("JWT_EXPIRATION_MINUTES not configured")
	}
	now := time.Now()
	jwtLogger.Debug("SignToken: expiration duration: %v, issued at: %v", expiration, now)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   userUUID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Use HS256 (HMAC-SHA256) - symmetric signing
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		jwtLogger.LogError("SignToken", err)
		jwtLogger.Error("SignToken: failed to sign token for userUUID: %s", userUUID)
		return "", err
	}

	jwtLogger.Debug("SignToken: successfully generated token for userUUID: %s tokenLength=%d", userUUID, len(tokenString))
	return tokenString, nil
}

// SignPendingRegistrationToken creates a short-lived JWT for pending registration
// This token contains the pending ID that references server-side stored AccountNumber
// Security: Short expiration from config, contains is_pending flag
func SignPendingRegistrationToken(pendingID string) (string, error) {
	jwtLogger.Debug("SignPendingRegistrationToken: starting for pendingID: %s", pendingID)

	secret, issuer, audience, err := getJWTBaseConfig()
	if err != nil {
		jwtLogger.Error("SignPendingRegistrationToken: %v", err)
		return "", err
	}

	now := time.Now()
	expiration := config.GetPendingTokenTTL()
	if expiration <= 0 {
		jwtLogger.Error("SignPendingRegistrationToken: invalid PENDING_TOKEN_TTL_SEC")
		return "", errors.New("PENDING_TOKEN_TTL_SEC not configured")
	}

	claims := PendingRegistrationClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   "pending_registration",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			NotBefore: jwt.NewNumericDate(now),
		},
		PendingID: pendingID,
		IsPending: true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		jwtLogger.LogError("SignPendingRegistrationToken", err)
		return "", err
	}

	jwtLogger.Debug("SignPendingRegistrationToken: successfully generated for pendingID: %s", pendingID)
	return tokenString, nil
}

// VerifyPendingRegistrationToken verifies a pending registration token and returns the pending AccountNumber
// Returns error if token is not a pending registration token or is invalid/expired
func VerifyPendingRegistrationToken(tokenString string) (string, error) {
	jwtLogger.Debug("VerifyPendingRegistrationToken: starting verification")

	secret, issuer, audience, err := getJWTBaseConfig()
	if err != nil {
		jwtLogger.Error("VerifyPendingRegistrationToken: %v", err)
		return "", err
	}

	token, err := jwt.ParseWithClaims(tokenString, &PendingRegistrationClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			jwtLogger.Warn("VerifyPendingRegistrationToken: unexpected signing method: %v", token.Method.Alg())
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			jwtLogger.Warn("VerifyPendingRegistrationToken: token expired")
			return "", ErrExpiredToken
		}
		jwtLogger.LogError("VerifyPendingRegistrationToken", err)
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*PendingRegistrationClaims)
	if !ok || !token.Valid {
		jwtLogger.Warn("VerifyPendingRegistrationToken: invalid claims")
		return "", ErrInvalidToken
	}

	// Verify this is a pending registration token
	if !claims.IsPending || claims.Subject != "pending_registration" {
		jwtLogger.Warn("VerifyPendingRegistrationToken: not a pending registration token")
		return "", ErrInvalidToken
	}

	if claims.PendingID == "" {
		jwtLogger.Warn("VerifyPendingRegistrationToken: pending ID is empty")
		return "", ErrInvalidToken
	}

	// Retrieve AccountNumber from server-side store
	// Note: GetPendingRegistration already deletes the entry (one-time use)
	accountNumber, err := GetPendingRegistration(claims.PendingID)
	if err != nil {
		jwtLogger.Warn("VerifyPendingRegistrationToken: failed to retrieve AccountNumber for pendingID (masked)")
		return "", err
	}

	jwtLogger.Debug("VerifyPendingRegistrationToken: verified pendingID and retrieved AccountNumber (masked)")
	return accountNumber, nil
}

// VerifyToken verifies a JWT token and returns the user UUID
// Security: Validates signing method, issuer, audience, and expiration
func VerifyToken(tokenString string) (string, error) {
	jwtLogger.Debug("VerifyToken: starting token verification, tokenLength=%d", len(tokenString))

	secret, issuer, audience, err := getJWTBaseConfig()
	if err != nil {
		jwtLogger.Error("VerifyToken: %v", err)
		return "", err
	}

	// Parse with explicit validation options
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// CRITICAL: Validate signing method to prevent algorithm confusion attacks
		// Only accept HMAC (HS256/HS384/HS512), reject RS256, ES256, none, etc.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			jwtLogger.Warn("VerifyToken: unexpected signing method: %v", token.Method.Alg())
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), // Explicitly allow only HS256
		jwt.WithIssuer(issuer),       // Validate issuer
		jwt.WithAudience(audience),   // Validate audience
		jwt.WithExpirationRequired(), // Require expiration
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			jwtLogger.Warn("VerifyToken: token expired")
			return "", ErrExpiredToken
		}
		jwtLogger.LogError("VerifyToken", err)
		jwtLogger.Warn("VerifyToken: invalid token")
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		jwtLogger.Warn("VerifyToken: token claims invalid or token not valid")
		return "", ErrInvalidToken
	}

	// Validate Subject is not empty
	if claims.Subject == "" {
		jwtLogger.Warn("VerifyToken: Subject claim is empty")
		return "", ErrInvalidToken
	}

	jwtLogger.Debug("VerifyToken: successfully verified token for userUUID: %s", claims.Subject)
	return claims.Subject, nil
}
