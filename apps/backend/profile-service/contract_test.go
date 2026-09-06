package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"libs/backend/shared/auth/authz"
)

// contractPath is the frozen document this service is built against. Reading it
// rather than restating it is what makes the suite below a drift check: an
// operation added, removed or re-authorized there fails here.
const contractPath = "../usersrole/docs/contracts/profile-service.openapi.yaml"

// operationCount is stated separately from the parsed set so that a scanner
// which silently stops matching cannot leave every comparison below passing
// vacuously against two empty sets.
const operationCount = 15

type contractOperation struct {
	operationID string
	method      string
	path        string
	rule        authz.Rule
}

func (o contractOperation) key() string { return o.method + " " + o.path }

var (
	contractPathLine   = regexp.MustCompile(`^ {2}(/\S*):$`)
	contractMethodLine = regexp.MustCompile(`^ {4}(get|put|post|delete|patch|head|options):$`)
	contractIDLine     = regexp.MustCompile(`^ {6}operationId: (\S+)$`)
	contractAuthLine   = regexp.MustCompile(`^ {6}x-authorization:$`)
	contractRuleLine   = regexp.MustCompile(`^ {8}rule: (\S+)$`)
)

// contractOperations scans the frozen document for its paths, methods and
// per-operation authorization rules. It reads the YAML by shape rather than
// through a parser so that this service takes no dependency on one; the shape
// it relies on is asserted by the count above.
func contractOperations(t *testing.T) []contractOperation {
	t.Helper()
	content, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read %s: %v", contractPath, err)
	}

	operations := []contractOperation{}
	inPaths := false
	path, method, operationID := "", "", ""
	inAuthorization := false

	flush := func() {
		if method != "" && operationID == "" {
			t.Fatalf("%s %s has no operationId", method, path)
		}
		method, operationID, inAuthorization = "", "", false
	}

	for _, line := range strings.Split(string(content), "\n") {
		switch {
		case line == "paths:":
			inPaths = true
			continue
		case !inPaths:
			continue
		case line != "" && !strings.HasPrefix(line, " "):
			// A new top-level key ends the paths section.
			flush()
			inPaths = false
			continue
		}

		if match := contractPathLine.FindStringSubmatch(line); match != nil {
			flush()
			path = match[1]
			continue
		}
		if match := contractMethodLine.FindStringSubmatch(line); match != nil {
			flush()
			method = strings.ToUpper(match[1])
			continue
		}
		if match := contractIDLine.FindStringSubmatch(line); match != nil {
			operationID = match[1]
			continue
		}
		if contractAuthLine.MatchString(line) {
			inAuthorization = true
			continue
		}
		if match := contractRuleLine.FindStringSubmatch(line); match != nil && inAuthorization {
			operations = append(operations, contractOperation{
				operationID: operationID, method: method, path: path, rule: authz.Rule(match[1]),
			})
			inAuthorization = false
		}
	}
	return operations
}

func TestTheContractStillDescribesFifteenAuthorizedOperations(t *testing.T) {
	operations := contractOperations(t)

	if len(operations) != operationCount {
		t.Fatalf("the contract names %d authorized operations, want %d: %v",
			len(operations), operationCount, operations)
	}
	for _, operation := range operations {
		if !strings.HasPrefix(operation.path, "/api/profiles") {
			t.Errorf("%s is not under /api/profiles", operation.key())
		}
		if operation.rule == "" {
			t.Errorf("%s names no authorization rule", operation.key())
		}
	}
}

func TestTheServedRouteSetIsExactlyTheContractsOperationSet(t *testing.T) {
	// The drift check the whole freeze rests on: a route this service serves
	// that the contract does not describe is as much a failure as one it
	// describes and this service does not serve.
	server, err := NewServer(ServerConfig{
		Store: stubStore{}, Verifier: parityVerifier(t), CORS: springShapedCORS(),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	served := map[string]authz.Rule{}
	for _, operation := range server.Operations() {
		served[operation.Method+" "+operation.Pattern] = operation.Rule
	}
	contract := map[string]authz.Rule{}
	for _, operation := range contractOperations(t) {
		contract[operation.key()] = operation.rule
	}

	if len(served) != operationCount {
		t.Errorf("the service serves %d operations, want %d", len(served), operationCount)
	}
	for key, rule := range contract {
		servedRule, present := served[key]
		if !present {
			t.Errorf("the contract describes %s and the service does not serve it", key)
			continue
		}
		if servedRule != rule {
			t.Errorf("%s is served under %s, the contract names %s", key, servedRule, rule)
		}
		delete(served, key)
	}
	for key := range served {
		t.Errorf("the service serves %s and the contract does not describe it", key)
	}
}

func TestTheParitySuiteDrivesEveryOperationTheContractDescribes(t *testing.T) {
	// Without this, an operation could be served and specified correctly and
	// still never have a request made against it.
	covered := map[string]authz.Rule{}
	for _, tc := range parityCases() {
		key := tc.method + " " + contractShape(tc.path)
		if _, twice := covered[key]; twice {
			t.Errorf("%s appears in the parity cases more than once", key)
		}
		covered[key] = tc.rule
	}

	for _, operation := range contractOperations(t) {
		rule, present := covered[operation.key()]
		if !present {
			t.Errorf("no parity case drives %s (%s)", operation.key(), operation.operationID)
			continue
		}
		if rule != operation.rule {
			t.Errorf("the parity case for %s uses %s, the contract names %s", operation.key(), rule, operation.rule)
		}
		delete(covered, operation.key())
	}
	for key := range covered {
		t.Errorf("a parity case drives %s, which the contract does not describe", key)
	}
}

func TestEveryRuleTheContractNamesHasAnAllowAndADenyCase(t *testing.T) {
	// A rule with no deny case would pass its allow cases with the check removed
	// altogether.
	for _, operation := range contractOperations(t) {
		if len(allowedBy(operation.rule)) == 0 {
			t.Errorf("rule %s (%s) has no principal that must pass it", operation.rule, operation.key())
		}
		if len(deniedBy(operation.rule)) == 0 {
			t.Errorf("rule %s (%s) has no principal that must fail it", operation.rule, operation.key())
		}
	}
}

func TestEveryRuleTheContractNamesIsOneTheSharedLibraryDecides(t *testing.T) {
	implemented := map[authz.Rule]bool{}
	for _, rule := range authz.Rules() {
		implemented[rule] = true
	}

	for _, operation := range contractOperations(t) {
		if !implemented[operation.rule] {
			t.Errorf("rule %s (%s) is not one the shared library decides", operation.rule, operation.key())
		}
	}
}

// contractShape turns a concrete request path back into the templated pattern
// the contract uses, so a parity case's path can be compared against it.
func contractShape(path string) string {
	segments := strings.Split(path, "/")
	if len(segments) < 4 {
		return path
	}
	if segments[3] == "by-user" {
		segments[4] = "{userId}"
		return strings.Join(segments, "/")
	}
	segments[3] = "{profileId}"
	if len(segments) == 6 && segments[4] == "address" {
		segments[5] = "{addressId}"
	}
	return strings.Join(segments, "/")
}

func TestTheContractsOperationIdsAreTheOnesTheJvmControllerDeclares(t *testing.T) {
	// A rename on either side is a client-visible change to the generated
	// clients, so the set is pinned rather than merely counted.
	want := []string{
		"addAddress", "addIcon", "createProfile", "deleteAddress", "deleteIcon",
		"deleteProfileById", "deleteProfileByUserId", "getProfileById", "getProfileByUserId",
		"getProfileIcon", "getProfiles", "updateAddress", "updateIcon", "updateProfileById",
		"updateProfileByUserId",
	}

	got := make([]string, 0, operationCount)
	for _, operation := range contractOperations(t) {
		got = append(got, operation.operationID)
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("operation ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("operation ids[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
