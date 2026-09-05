// Package auth verifies the HS256 tokens the JVM identity service mints and
// turns their claims into a principal the authz package can decide on.
//
// It is the only place either Go service parses a token, so the two cannot
// drift apart from each other or from the frozen service contracts.
package auth

import "time"

// Claim names, exactly as JwtService.generateToken writes them.
const (
	ClaimSubject   = "sub"
	ClaimRoles     = "roles"
	ClaimUserID    = "user_id"
	ClaimProfileID = "profile_id"
	ClaimIssuer    = "iss"
	ClaimAudience  = "aud"
	ClaimTokenID   = "jti"
	ClaimIssuedAt  = "iat"
	ClaimNotBefore = "nbf"
	ClaimExpiresAt = "exp"
)

// Principal is the verified caller, standing in for the JVM's SecurityUser.
//
// Every field is frozen at mint time. SecurityUser rehydrates the user from the
// database on each request, so a principal built here can be staler than the
// one the JVM would build for the same token — bounded by the token lifetime
// for roles, and unbounded for a profile the caller has since deleted. Both
// divergences are recorded in the frozen contracts.
type Principal struct {
	Subject string
	Roles   []string
	// UserID and ProfileID are pointers because the claims are nullable: the
	// JVM writes JSON null for a principal with no profile row, and an absent
	// or null ProfileID is what makes the profile fallback engage.
	UserID    *int64
	ProfileID *int64
	Issuer    string
	Audience  []string
	TokenID   string
	IssuedAt  time.Time
	NotBefore time.Time
	ExpiresAt time.Time
}

// EmailAddress mirrors SecurityUser.getUsername(), which returns the email
// address rather than a user name. Rules that compare against the subject read
// better through this name.
func (p *Principal) EmailAddress() string {
	return p.Subject
}

// HasAuthority reports whether the principal was granted the named role.
// The comparison is exact, matching SimpleGrantedAuthority equality.
func (p *Principal) HasAuthority(name string) bool {
	for _, role := range p.Roles {
		if role == name {
			return true
		}
	}
	return false
}

// HasAnyAuthority mirrors the hasAnyAuthority predicate.
func (p *Principal) HasAnyAuthority(names ...string) bool {
	for _, name := range names {
		if p.HasAuthority(name) {
			return true
		}
	}
	return false
}
