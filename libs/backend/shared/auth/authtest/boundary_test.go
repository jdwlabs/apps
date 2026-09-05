package authtest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The minter exists so tests need no running identity service. Nothing that
// ships may import it: it signs tokens, and a service that can sign its own
// tokens can mint itself any principal it likes, which is the whole
// authorization model gone.
//
// Go has no visibility mechanism that expresses "test code only" across
// modules, and a build tag would hide the package from the tests that need it.
// So the rule is asserted rather than declared: this walks the service modules
// and fails on any non-test file importing it.
const minterPackage = "libs/backend/shared/auth/authtest"

func TestNoServiceImportsTheMinter(t *testing.T) {
	// From libs/backend/shared/auth/authtest to the repository root.
	root := filepath.Join("..", "..", "..", "..", "..")
	services := filepath.Join(root, "apps", "backend")
	if _, err := os.Stat(services); err != nil {
		t.Fatalf("cannot reach the service modules to check them: %v", err)
	}

	var offenders []string
	err := filepath.WalkDir(services, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// Vendored or generated trees would only produce noise.
			if entry.Name() == "vendor" || entry.Name() == "build" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(content), `"`+minterPackage+`"`) {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk the service modules: %v", err)
	}

	for _, offender := range offenders {
		t.Errorf("%s imports %s outside a test; it can mint tokens for any principal", offender, minterPackage)
	}
}
