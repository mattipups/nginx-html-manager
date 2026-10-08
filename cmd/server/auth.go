package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

type AuthUser struct {
	Username string   `json:"username"`
	Role     Role     `json:"role"`
	Groups   []string `json:"groups,omitempty"`
	AuthType string   `json:"authType"`
}

func (u *AuthUser) CanUpload() bool {
	return u.Role == RoleAdmin || u.Role == RoleEditor
}

func (u *AuthUser) CanUpdate() bool {
	return u.Role == RoleAdmin || u.Role == RoleEditor
}

func (u *AuthUser) CanDelete() bool {
	return u.Role == RoleAdmin
}

func (u *AuthUser) CanViewAudit() bool {
	return u.Role == RoleAdmin
}

func (u *AuthUser) CanManageQuota() bool {
	return u.Role == RoleAdmin
}

func (a *app) authenticate(r *http.Request) (*AuthUser, bool) {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if oidcUser, ok := a.validateOIDCToken(token); ok {
			return oidcUser, true
		}
	}

	user, pass, ok := r.BasicAuth()
	if !ok {
		return nil, false
	}

	validUser := equal(user, a.user)
	validPass := equal(pass, a.password)
	if !validUser || !validPass {
		return nil, false
	}

	return &AuthUser{
		Username: user,
		Role:     RoleAdmin,
		Groups:   []string{"administrators"},
		AuthType: "basic",
	}, true
}

func (a *app) validateOIDCToken(token string) (*AuthUser, bool) {
	if env("OIDC_ENABLED", "false") != "true" {
		return nil, false
	}

	oidcSecret := env("OIDC_CLIENT_SECRET", "")
	if oidcSecret == "" || subtle.ConstantTimeCompare([]byte(token), []byte(oidcSecret)) != 1 {
		return nil, false
	}

	role := Role(env("OIDC_DEFAULT_ROLE", "viewer"))
	if role != RoleAdmin && role != RoleEditor && role != RoleViewer {
		role = RoleViewer
	}

	return &AuthUser{
		Username: env("OIDC_TEST_USER", "oidc-user"),
		Role:     role,
		Groups:   strings.Split(env("OIDC_USER_GROUPS", "users"), ","),
		AuthType: "oidc",
	}, true
}
