// Package authz decides the authorization rules the frozen service contracts
// name, one function per rule, over a verified principal and the ids the
// request carries.
//
// Each rule reproduces a @PreAuthorize predicate from the Spring application it
// replaces. The predicate each one transcribes is quoted on the constant, so a
// reviewer can compare the two without leaving the file.
package authz

import (
	"context"
	"errors"
	"fmt"

	"libs/backend/shared/auth"
)

// Authority names, matching the role names the JVM turns into
// SimpleGrantedAuthority instances.
const (
	AuthorityAdmin   = "ADMIN"
	AuthorityManager = "MANAGER"
)

// Rule names an authorization rule. The values are the x-authorization.rule
// strings the frozen contracts carry, so an operation's specification names its
// rule and a handler passes that same name here.
type Rule string

const (
	// RulePublic is reachable without a token, through the filter chain's
	// permitAll matchers rather than through any method predicate.
	RulePublic Rule = "PUBLIC"
	// RuleAuthenticated carries no method predicate: any verified principal passes.
	RuleAuthenticated Rule = "AUTHENTICATED"
	// RuleAdmin transcribes hasAuthority('ADMIN').
	RuleAdmin Rule = "ADMIN"
	// RuleAdminOrManager transcribes hasAnyAuthority('ADMIN', 'MANAGER').
	RuleAdminOrManager Rule = "ADMIN_OR_MANAGER"
	// RuleAdminOrSelfByUserID transcribes
	// hasAuthority('ADMIN') or #userId == authentication.principal.getUserId().
	RuleAdminOrSelfByUserID Rule = "ADMIN_OR_SELF_BY_USER_ID"
	// RuleAdminOrSelfByEmail transcribes
	// hasAuthority('ADMIN') or #emailAddress == authentication.principal.getUsername().
	RuleAdminOrSelfByEmail Rule = "ADMIN_OR_SELF_BY_EMAIL"
	// RuleAdminOrSelfByProfileID transcribes
	// hasAuthority('ADMIN') or #profileId == authentication.principal.getProfileId(),
	// with the profile fallback the contract requires.
	RuleAdminOrSelfByProfileID Rule = "ADMIN_OR_SELF_BY_PROFILE_ID"
	// RuleAdminOrSelfByBodyUserID transcribes
	// hasAuthority('ADMIN') or #profile.userId() == authentication.principal.getUserId().
	// The only rule that reads the request body rather than a path variable.
	RuleAdminOrSelfByBodyUserID Rule = "ADMIN_OR_SELF_BY_BODY_USER_ID"
)

var (
	ErrUnknownRule    = errors.New("unknown authorization rule")
	ErrMissingSubject = errors.New("the rule compares an id the request did not supply")
)

// Rules lists every rule this package decides.
func Rules() []Rule {
	return []Rule{
		RulePublic,
		RuleAuthenticated,
		RuleAdmin,
		RuleAdminOrManager,
		RuleAdminOrSelfByUserID,
		RuleAdminOrSelfByEmail,
		RuleAdminOrSelfByProfileID,
		RuleAdminOrSelfByBodyUserID,
	}
}

// Subject holds the ids a rule compares the principal against. Each is a
// pointer so that "the request did not carry this" is distinguishable from
// zero, which is a valid id in no schema but a silent allow in a comparison.
type Subject struct {
	UserID       *int64
	ProfileID    *int64
	EmailAddress *string
	// BodyUserID comes from the request body rather than the path, and is kept
	// separate from UserID so a handler cannot satisfy a path-scoped rule with
	// a value the caller chose.
	BodyUserID *int64
}

// ProfileIDResolver reads the caller's profile id from storage. It reports
// found=false when the caller owns no profile.
type ProfileIDResolver func(ctx context.Context, userID int64) (profileID int64, found bool, err error)

// Authorizer decides rules. The zero value decides every rule except the
// profile fallback, which needs storage access a service must supply.
type Authorizer struct {
	// ProfileIDForUser closes the lockout the split would otherwise introduce:
	// the profile_id claim is stamped at login and never refreshed, so a caller
	// who creates a profile mid-session carries a null claim for the rest of
	// the token's life. It is consulted only when the claim is absent, and it
	// keys on the user_id claim rather than on anything the request supplies,
	// so falling back cannot widen the caller's own authorization.
	//
	// Leaving it nil denies profile-scoped access to a caller with no claim,
	// which is correct for a service that owns no profile storage.
	ProfileIDForUser ProfileIDResolver
}

// Allow decides one rule. It is the entry point a handler or middleware uses;
// the exported per-rule functions exist for call sites that know their rule
// statically and want the compiler to check their arguments.
func (a Authorizer) Allow(ctx context.Context, rule Rule, p *auth.Principal, s Subject) (bool, error) {
	switch rule {
	case RulePublic:
		return true, nil
	case RuleAuthenticated:
		return Authenticated(p), nil
	case RuleAdmin:
		return Admin(p), nil
	case RuleAdminOrManager:
		return AdminOrManager(p), nil
	case RuleAdminOrSelfByUserID:
		userID, err := require(s.UserID, rule, "user id")
		if err != nil {
			return false, err
		}
		return AdminOrSelfByUserID(p, userID), nil
	case RuleAdminOrSelfByBodyUserID:
		userID, err := require(s.BodyUserID, rule, "body user id")
		if err != nil {
			return false, err
		}
		return AdminOrSelfByBodyUserID(p, userID), nil
	case RuleAdminOrSelfByEmail:
		email, err := require(s.EmailAddress, rule, "email address")
		if err != nil {
			return false, err
		}
		return AdminOrSelfByEmail(p, email), nil
	case RuleAdminOrSelfByProfileID:
		profileID, err := require(s.ProfileID, rule, "profile id")
		if err != nil {
			return false, err
		}
		return a.AdminOrSelfByProfileID(ctx, p, profileID)
	default:
		return false, fmt.Errorf("%w: %s", ErrUnknownRule, rule)
	}
}

// Authenticated passes any verified principal.
func Authenticated(p *auth.Principal) bool { return p != nil }

// Admin decides hasAuthority('ADMIN').
func Admin(p *auth.Principal) bool {
	return p != nil && p.HasAuthority(AuthorityAdmin)
}

// AdminOrManager decides hasAnyAuthority('ADMIN', 'MANAGER').
func AdminOrManager(p *auth.Principal) bool {
	return p != nil && p.HasAnyAuthority(AuthorityAdmin, AuthorityManager)
}

// AdminOrSelfByUserID decides the path-scoped self check against the user_id claim.
func AdminOrSelfByUserID(p *auth.Principal, userID int64) bool {
	return Admin(p) || (p != nil && p.UserID != nil && *p.UserID == userID)
}

// AdminOrSelfByBodyUserID decides the self check against a user id the request
// body carries.
func AdminOrSelfByBodyUserID(p *auth.Principal, userID int64) bool {
	return AdminOrSelfByUserID(p, userID)
}

// AdminOrSelfByEmail decides the self check against the token subject. The
// comparison is exact and case-sensitive, so a caller authenticated as
// User@example.com cannot read user@example.com.
func AdminOrSelfByEmail(p *auth.Principal, emailAddress string) bool {
	return Admin(p) || (p != nil && p.EmailAddress() == emailAddress)
}

// AdminOrSelfByProfileID decides the profile-scoped self check, falling back to
// a storage lookup when the claim is absent.
//
// A present claim is used as-is even when the profile it names is gone: the
// contract records that widening deliberately, because re-checking existence
// here would reinstate the per-request database read the split removes.
func (a Authorizer) AdminOrSelfByProfileID(ctx context.Context, p *auth.Principal, profileID int64) (bool, error) {
	if Admin(p) {
		return true, nil
	}
	if p == nil {
		return false, nil
	}
	if p.ProfileID != nil {
		return *p.ProfileID == profileID, nil
	}
	if a.ProfileIDForUser == nil || p.UserID == nil {
		return false, nil
	}
	resolved, found, err := a.ProfileIDForUser(ctx, *p.UserID)
	if err != nil {
		return false, fmt.Errorf("resolve profile for user %d: %w", *p.UserID, err)
	}
	if !found {
		return false, nil
	}
	return resolved == profileID, nil
}

func require[T any](value *T, rule Rule, name string) (T, error) {
	if value == nil {
		var zero T
		return zero, fmt.Errorf("%w: %s needs a %s", ErrMissingSubject, rule, name)
	}
	return *value, nil
}
