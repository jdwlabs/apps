package authz_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authz"
)

func ptr[T any](v T) *T { return &v }

func principal(email string, roles []string, userID *int64, profileID *int64) *auth.Principal {
	return &auth.Principal{Subject: email, Roles: roles, UserID: userID, ProfileID: profileID}
}

var (
	admin     = principal("admin@jdw.com", []string{"ADMIN"}, ptr(int64(1)), ptr(int64(10)))
	manager   = principal("manager@jdw.com", []string{"MANAGER"}, ptr(int64(2)), ptr(int64(20)))
	plain     = principal("user@jdw.com", []string{"USER"}, ptr(int64(3)), ptr(int64(30)))
	noProfile = principal("fresh@jdw.com", []string{"USER"}, ptr(int64(4)), nil)
	noUser    = principal("broken@jdw.com", []string{"USER"}, nil, nil)
)

// resolverReturning stands in for the indexed read on auth.profiles.user_id
// that profile-service performs when the profile_id claim is absent.
func resolverReturning(profileID int64, found bool, err error) func(context.Context, int64) (int64, bool, error) {
	return func(context.Context, int64) (int64, bool, error) { return profileID, found, err }
}

func TestAllowCoversEveryContractRule(t *testing.T) {
	withFallback := authz.Authorizer{ProfileIDForUser: resolverReturning(40, true, nil)}
	noFallback := authz.Authorizer{}

	tests := []struct {
		name       string
		authorizer authz.Authorizer
		rule       authz.Rule
		principal  *auth.Principal
		subject    authz.Subject
		want       bool
	}{
		{"public allows an anonymous caller", noFallback, authz.RulePublic, nil, authz.Subject{}, true},
		{"public allows an authenticated caller", noFallback, authz.RulePublic, plain, authz.Subject{}, true},

		{"authenticated allows any principal", noFallback, authz.RuleAuthenticated, plain, authz.Subject{}, true},
		{"authenticated allows an admin", noFallback, authz.RuleAuthenticated, admin, authz.Subject{}, true},
		{"authenticated denies an anonymous caller", noFallback, authz.RuleAuthenticated, nil, authz.Subject{}, false},

		{"admin allows an admin", noFallback, authz.RuleAdmin, admin, authz.Subject{}, true},
		{"admin denies a manager", noFallback, authz.RuleAdmin, manager, authz.Subject{}, false},
		{"admin denies a plain user", noFallback, authz.RuleAdmin, plain, authz.Subject{}, false},
		{"admin denies an anonymous caller", noFallback, authz.RuleAdmin, nil, authz.Subject{}, false},

		{"admin or manager allows an admin", noFallback, authz.RuleAdminOrManager, admin, authz.Subject{}, true},
		{"admin or manager allows a manager", noFallback, authz.RuleAdminOrManager, manager, authz.Subject{}, true},
		{"admin or manager denies a plain user", noFallback, authz.RuleAdminOrManager, plain, authz.Subject{}, false},
		{"admin or manager denies an anonymous caller", noFallback, authz.RuleAdminOrManager, nil, authz.Subject{}, false},

		{"self by user id allows an admin over anyone", noFallback, authz.RuleAdminOrSelfByUserID, admin, authz.Subject{UserID: ptr(int64(999))}, true},
		{"self by user id allows the owner", noFallback, authz.RuleAdminOrSelfByUserID, plain, authz.Subject{UserID: ptr(int64(3))}, true},
		{"self by user id denies another user", noFallback, authz.RuleAdminOrSelfByUserID, plain, authz.Subject{UserID: ptr(int64(999))}, false},
		{"self by user id gives a manager no privilege", noFallback, authz.RuleAdminOrSelfByUserID, manager, authz.Subject{UserID: ptr(int64(999))}, false},
		{"self by user id denies a principal with no user claim", noFallback, authz.RuleAdminOrSelfByUserID, noUser, authz.Subject{UserID: ptr(int64(3))}, false},
		{"self by user id denies an anonymous caller", noFallback, authz.RuleAdminOrSelfByUserID, nil, authz.Subject{UserID: ptr(int64(3))}, false},

		{"self by email allows an admin over anyone", noFallback, authz.RuleAdminOrSelfByEmail, admin, authz.Subject{EmailAddress: ptr("someone@jdw.com")}, true},
		{"self by email allows the owner", noFallback, authz.RuleAdminOrSelfByEmail, plain, authz.Subject{EmailAddress: ptr("user@jdw.com")}, true},
		{"self by email is case sensitive", noFallback, authz.RuleAdminOrSelfByEmail, plain, authz.Subject{EmailAddress: ptr("User@jdw.com")}, false},
		{"self by email denies another address", noFallback, authz.RuleAdminOrSelfByEmail, plain, authz.Subject{EmailAddress: ptr("other@jdw.com")}, false},
		{"self by email denies an anonymous caller", noFallback, authz.RuleAdminOrSelfByEmail, nil, authz.Subject{EmailAddress: ptr("user@jdw.com")}, false},

		{"self by body user id allows an admin over anyone", noFallback, authz.RuleAdminOrSelfByBodyUserID, admin, authz.Subject{BodyUserID: ptr(int64(999))}, true},
		{"self by body user id allows the owner", noFallback, authz.RuleAdminOrSelfByBodyUserID, plain, authz.Subject{BodyUserID: ptr(int64(3))}, true},
		{"self by body user id denies another user", noFallback, authz.RuleAdminOrSelfByBodyUserID, plain, authz.Subject{BodyUserID: ptr(int64(999))}, false},
		{"self by body user id denies an anonymous caller", noFallback, authz.RuleAdminOrSelfByBodyUserID, nil, authz.Subject{BodyUserID: ptr(int64(3))}, false},

		{"self by profile id allows an admin over anyone", withFallback, authz.RuleAdminOrSelfByProfileID, admin, authz.Subject{ProfileID: ptr(int64(999))}, true},
		{"self by profile id allows the claim owner", withFallback, authz.RuleAdminOrSelfByProfileID, plain, authz.Subject{ProfileID: ptr(int64(30))}, true},
		{"self by profile id denies another profile", withFallback, authz.RuleAdminOrSelfByProfileID, plain, authz.Subject{ProfileID: ptr(int64(999))}, false},
		{"self by profile id falls back when the claim is absent", withFallback, authz.RuleAdminOrSelfByProfileID, noProfile, authz.Subject{ProfileID: ptr(int64(40))}, true},
		{"self by profile id denies when the fallback disagrees", withFallback, authz.RuleAdminOrSelfByProfileID, noProfile, authz.Subject{ProfileID: ptr(int64(999))}, false},
		{"self by profile id denies when no fallback is configured", noFallback, authz.RuleAdminOrSelfByProfileID, noProfile, authz.Subject{ProfileID: ptr(int64(40))}, false},
		{"self by profile id denies a principal with no user claim to fall back on", withFallback, authz.RuleAdminOrSelfByProfileID, noUser, authz.Subject{ProfileID: ptr(int64(40))}, false},
		{"self by profile id denies an anonymous caller", withFallback, authz.RuleAdminOrSelfByProfileID, nil, authz.Subject{ProfileID: ptr(int64(40))}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.authorizer.Allow(context.Background(), tc.rule, tc.principal, tc.subject)
			if err != nil {
				t.Fatalf("Allow: %v", err)
			}
			if got != tc.want {
				t.Errorf("Allow(%s) = %v, want %v", tc.rule, got, tc.want)
			}
		})
	}
}

func TestAllowKnowsEveryRuleTheFrozenContractsName(t *testing.T) {
	// The contracts are the specification this library exists to reproduce, so
	// a rule added to one of them has to fail here rather than be discovered by
	// a service author reading YAML.
	contractRules := rulesInFrozenContracts(t)
	if len(contractRules) == 0 {
		t.Fatal("no x-authorization rules found in the frozen contracts")
	}

	implemented := map[authz.Rule]bool{}
	for _, rule := range authz.Rules() {
		implemented[rule] = true
	}

	for _, name := range contractRules {
		if !implemented[authz.Rule(name)] {
			t.Errorf("contract rule %s has no implementation", name)
		}
		delete(implemented, authz.Rule(name))
	}
	for rule := range implemented {
		t.Errorf("rule %s is implemented but no operation in either contract uses it", rule)
	}
}

func TestAllowRejectsAnUnknownRule(t *testing.T) {
	_, err := authz.Authorizer{}.Allow(context.Background(), authz.Rule("ADMIN_OR_WHATEVER"), admin, authz.Subject{})

	if !errors.Is(err, authz.ErrUnknownRule) {
		t.Errorf("error = %v, want %v", err, authz.ErrUnknownRule)
	}
}

func TestAllowRequiresTheIdTheRuleCompares(t *testing.T) {
	tests := []struct {
		name string
		rule authz.Rule
	}{
		{"user id", authz.RuleAdminOrSelfByUserID},
		{"profile id", authz.RuleAdminOrSelfByProfileID},
		{"email address", authz.RuleAdminOrSelfByEmail},
		{"body user id", authz.RuleAdminOrSelfByBodyUserID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// A plain user rather than an admin: an admin short-circuits before
			// the comparison, which would hide the missing id.
			_, err := authz.Authorizer{}.Allow(context.Background(), tc.rule, plain, authz.Subject{})
			if !errors.Is(err, authz.ErrMissingSubject) {
				t.Errorf("error = %v, want %v", err, authz.ErrMissingSubject)
			}
		})
	}
}

func TestProfileFallbackIsSkippedWhenTheClaimIsPresent(t *testing.T) {
	called := false
	a := authz.Authorizer{ProfileIDForUser: func(context.Context, int64) (int64, bool, error) {
		called = true
		return 40, true, nil
	}}

	got, err := a.Allow(context.Background(), authz.RuleAdminOrSelfByProfileID, plain, authz.Subject{ProfileID: ptr(int64(40))})
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}

	if called {
		t.Error("the fallback lookup ran; a present claim must short-circuit it so the common request pays nothing")
	}
	if got {
		t.Error("Allow = true; the present claim says profile 30, not 40")
	}
}

func TestProfileFallbackKeysOnTheUserClaimNotThePath(t *testing.T) {
	var askedFor int64
	a := authz.Authorizer{ProfileIDForUser: func(_ context.Context, userID int64) (int64, bool, error) {
		askedFor = userID
		return 40, true, nil
	}}

	if _, err := a.Allow(context.Background(), authz.RuleAdminOrSelfByProfileID, noProfile, authz.Subject{ProfileID: ptr(int64(40))}); err != nil {
		t.Fatalf("Allow: %v", err)
	}

	if askedFor != 4 {
		t.Errorf("fallback looked up user %d, want 4 from the user_id claim; keying on anything else lets a caller widen their own authorization", askedFor)
	}
}

func TestProfileFallbackDeniesWhenNoProfileExists(t *testing.T) {
	a := authz.Authorizer{ProfileIDForUser: resolverReturning(0, false, nil)}

	got, err := a.Allow(context.Background(), authz.RuleAdminOrSelfByProfileID, noProfile, authz.Subject{ProfileID: ptr(int64(40))})
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}

	if got {
		t.Error("Allow = true for a caller who owns no profile")
	}
}

func TestProfileFallbackSurfacesLookupFailures(t *testing.T) {
	lookupFailed := errors.New("connection refused")
	a := authz.Authorizer{ProfileIDForUser: resolverReturning(0, false, lookupFailed)}

	got, err := a.Allow(context.Background(), authz.RuleAdminOrSelfByProfileID, noProfile, authz.Subject{ProfileID: ptr(int64(40))})

	if !errors.Is(err, lookupFailed) {
		t.Errorf("error = %v, want it to wrap %v; a failed lookup is not a denial", err, lookupFailed)
	}
	if got {
		t.Error("Allow = true despite a failed lookup")
	}
}

// A claim pointing at a profile the caller has since deleted still authorizes.
// The frozen contract records this as a deliberate widening rather than a bug,
// so it is pinned here: restoring the JVM's 403 would need the per-request
// database read the split exists to remove.
func TestStaleProfileClaimStillAuthorizes(t *testing.T) {
	deleted := principal("gone@jdw.com", []string{"USER"}, ptr(int64(5)), ptr(int64(50)))
	a := authz.Authorizer{ProfileIDForUser: resolverReturning(0, false, nil)}

	got, err := a.Allow(context.Background(), authz.RuleAdminOrSelfByProfileID, deleted, authz.Subject{ProfileID: ptr(int64(50))})
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}

	if !got {
		t.Error("Allow = false; a present but stale claim is documented to pass, and changing that is a contract change")
	}
}

func TestRuleFunctionsMatchTheDispatcher(t *testing.T) {
	if !authz.Admin(admin) || authz.Admin(plain) {
		t.Error("Admin does not match hasAuthority('ADMIN')")
	}
	if !authz.AdminOrManager(manager) || authz.AdminOrManager(plain) {
		t.Error("AdminOrManager does not match hasAnyAuthority('ADMIN', 'MANAGER')")
	}
	if !authz.Authenticated(plain) || authz.Authenticated(nil) {
		t.Error("Authenticated does not match an authenticated principal")
	}
	if !authz.AdminOrSelfByUserID(plain, 3) || authz.AdminOrSelfByUserID(plain, 4) {
		t.Error("AdminOrSelfByUserID does not match #userId == principal.getUserId()")
	}
	if !authz.AdminOrSelfByEmail(plain, "user@jdw.com") || authz.AdminOrSelfByEmail(plain, "other@jdw.com") {
		t.Error("AdminOrSelfByEmail does not match #emailAddress == principal.getUsername()")
	}
	if !authz.AdminOrSelfByBodyUserID(plain, 3) || authz.AdminOrSelfByBodyUserID(plain, 4) {
		t.Error("AdminOrSelfByBodyUserID does not match #profile.userId() == principal.getUserId()")
	}
}

func TestAuthorityNamesMatchTheJvm(t *testing.T) {
	if authz.AuthorityAdmin != "ADMIN" {
		t.Errorf("AuthorityAdmin = %q, want ADMIN", authz.AuthorityAdmin)
	}
	if authz.AuthorityManager != "MANAGER" {
		t.Errorf("AuthorityManager = %q, want MANAGER", authz.AuthorityManager)
	}
}

var (
	authorizationBlockPattern = regexp.MustCompile(`^(\s+)x-authorization:\s*$`)
	rulePattern               = regexp.MustCompile(`^\s+rule:\s*(\S+)\s*$`)
)

// rulesInBlocks reads the rule of each x-authorization block, rather than every
// "rule:" key in the document: a later contract could add that key under some
// other extension, and matching it there would silently widen what the caller
// believes it has asserted.
//
// Scanned by indentation rather than by one regexp because RE2 has no
// backreferences, so the block's own indent cannot be matched against itself.
func rulesInBlocks(content []byte) []string {
	var rules []string
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		block := authorizationBlockPattern.FindStringSubmatch(line)
		if block == nil {
			continue
		}
		indent := len(block[1])
		for _, inner := range lines[i+1:] {
			if strings.TrimSpace(inner) == "" {
				continue
			}
			// The block ends at the first line indented no further than it.
			if len(inner)-len(strings.TrimLeft(inner, " ")) <= indent {
				break
			}
			if rule := rulePattern.FindStringSubmatch(inner); rule != nil {
				rules = append(rules, rule[1])
				break
			}
		}
	}
	return rules
}

func rulesInFrozenContracts(t *testing.T) []string {
	t.Helper()
	contracts := filepath.Join("..", "..", "..", "..", "..", "apps", "backend", "usersrole", "docs", "contracts")
	seen := map[string]bool{}
	for _, name := range []string{"identity-service.openapi.yaml", "profile-service.openapi.yaml"} {
		content, err := os.ReadFile(filepath.Join(contracts, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		found := rulesInBlocks(content)
		if len(found) == 0 {
			t.Fatalf("%s: no x-authorization rules matched; the contract layout changed", name)
		}
		for _, rule := range found {
			seen[rule] = true
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestRulesInBlocksReadsOnlyTheAuthorizationBlock(t *testing.T) {
	document := []byte(`paths:
  /api/users:
    get:
      x-authorization:
        rule: ADMIN
        preAuthorize: "hasAuthority('ADMIN')"
      x-some-later-extension:
        rule: NOT_AN_AUTHORIZATION_RULE
      responses:
        '200':
          description: ok
    post:
      x-authorization:
        note: |
          A block whose rule is not the first key.
        rule: AUTHENTICATED
`)

	got := rulesInBlocks(document)

	want := []string{"ADMIN", "AUTHENTICATED"}
	if len(got) != len(want) {
		t.Fatalf("rules = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rules[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTheContractsCarryEveryRuleTheLibraryImplements(t *testing.T) {
	// A count, so that a scanner change which silently stops matching cannot
	// leave TestAllowKnowsEveryRuleTheFrozenContractsName passing vacuously.
	if got, want := len(rulesInFrozenContracts(t)), len(authz.Rules()); got != want {
		t.Errorf("the contracts name %d distinct rules, the library implements %d", got, want)
	}
}
