/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/FallenProjects/telegram-web-player
 */

package webapp

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	telegramJWKSURL  = "https://oauth.telegram.org/.well-known/jwks.json"
	telegramIssuer   = "https://oauth.telegram.org"
	telegramClientID = "8501197173"
)

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

var (
	jwksCache     *JWKS
	jwksFetchTime time.Time
	jwksMu        sync.RWMutex
)

func getJWKS() (*JWKS, error) {
	jwksMu.RLock()
	if jwksCache != nil && time.Since(jwksFetchTime) < 1*time.Hour {
		defer jwksMu.RUnlock()
		return jwksCache, nil
	}
	jwksMu.RUnlock()

	jwksMu.Lock()
	defer jwksMu.Unlock()

	if jwksCache != nil && time.Since(jwksFetchTime) < 1*time.Hour {
		return jwksCache, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(telegramJWKSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Telegram JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	jwksCache = &jwks
	jwksFetchTime = time.Now()
	return jwksCache, nil
}

type JWTHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

type TelegramIDTokenClaims struct {
	Iss             string      `json:"iss"`
	Sub             json.Number `json:"sub"`
	Aud             any         `json:"aud"`
	Exp             int64       `json:"exp"`
	Iat             int64       `json:"iat"`
	Nonce           string      `json:"nonce,omitempty"`
	Name            string      `json:"name,omitempty"`
	PreferredName   string      `json:"preferred_username,omitempty"`
	GivenName       string      `json:"given_name,omitempty"`
	FamilyName      string      `json:"family_name,omitempty"`
	Picture         string      `json:"picture,omitempty"`
	Scope           string      `json:"scope,omitempty"`
	AllowsWriteToPM bool        `json:"allows_write_to_pm,omitempty"`
}

func parseBase64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func verifyTelegramIDToken(idToken string) (*WebAppUser, bool, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, false, errors.New("invalid JWT format")
	}

	headerBytes, err := parseBase64URL(parts[0])
	if err != nil {
		return nil, false, fmt.Errorf("invalid header base64: %w", err)
	}

	var header JWTHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, false, fmt.Errorf("invalid header json: %w", err)
	}

	payloadBytes, err := parseBase64URL(parts[1])
	if err != nil {
		return nil, false, fmt.Errorf("invalid payload base64: %w", err)
	}

	var claims TelegramIDTokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, false, fmt.Errorf("invalid claims json: %w", err)
	}

	now := time.Now().Unix()
	if claims.Exp != 0 && now > claims.Exp+300 { // 5m grace
		return nil, false, errors.New("token expired")
	}

	if claims.Iss != telegramIssuer {
		return nil, false, fmt.Errorf("invalid issuer: %s", claims.Iss)
	}

	audValid := false
	switch a := claims.Aud.(type) {
	case string:
		if a == telegramClientID {
			audValid = true
		}
	case []any:
		for _, item := range a {
			if str, ok := item.(string); ok && str == telegramClientID {
				audValid = true
				break
			}
		}
	}
	if !audValid {
		return nil, false, errors.New("invalid audience")
	}

	jwks, err := getJWKS()
	if err != nil {
		return nil, false, err
	}

	var matchedKey *JWK
	for _, key := range jwks.Keys {
		if key.Kid == header.Kid {
			matchedKey = &key
			break
		}
	}

	if matchedKey == nil {
		return nil, false, fmt.Errorf("public key with kid '%s' not found", header.Kid)
	}

	signatureBytes, err := parseBase64URL(parts[2])
	if err != nil {
		return nil, false, fmt.Errorf("invalid signature base64: %w", err)
	}

	signedData := []byte(parts[0] + "." + parts[1])

	switch matchedKey.Alg {
	case "RS256":
		nBytes, err := parseBase64URL(matchedKey.N)
		if err != nil {
			return nil, false, err
		}
		eBytes, err := parseBase64URL(matchedKey.E)
		if err != nil {
			return nil, false, err
		}

		e := 0
		for _, b := range eBytes {
			e = (e << 8) | int(b)
		}

		pubKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}

		hasher := sha256.New()
		hasher.Write(signedData)
		hashed := hasher.Sum(nil)

		if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed, signatureBytes); err != nil {
			return nil, false, fmt.Errorf("RSA signature verification failed: %w", err)
		}

	case "EdDSA":
		xBytes, err := parseBase64URL(matchedKey.X)
		if err != nil || len(xBytes) != ed25519.PublicKeySize {
			return nil, false, errors.New("invalid Ed25519 key")
		}

		pubKey := ed25519.PublicKey(xBytes)
		if !ed25519.Verify(pubKey, signedData, signatureBytes) {
			return nil, false, errors.New("Ed25519 signature verification failed")
		}

	default:
		return nil, false, fmt.Errorf("unsupported algorithm: %s", matchedKey.Alg)
	}

	userID, err := claims.Sub.Int64()
	if err != nil || userID <= 0 {
		subStr := claims.Sub.String()
		fmt.Sscanf(subStr, "%d", &userID)
	}
	if userID <= 0 {
		return nil, false, fmt.Errorf("invalid user ID in sub claim: %s", claims.Sub.String())
	}

	allowsWrite := claims.AllowsWriteToPM
	if !allowsWrite && claims.Scope != "" {
		scopes := strings.Fields(claims.Scope)
		if slices.Contains(scopes, "write") {
			allowsWrite = true
		}
	}

	user := &WebAppUser{
		ID:              userID,
		FirstName:       claims.GivenName,
		LastName:        claims.FamilyName,
		Username:        claims.PreferredName,
		PhotoURL:        claims.Picture,
		AllowsWriteToPM: allowsWrite,
	}
	if user.FirstName == "" && claims.Name != "" {
		user.FirstName = claims.Name
	}

	return user, allowsWrite, nil
}
