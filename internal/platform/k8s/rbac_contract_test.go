package k8s

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// RBAC contract: every typed client-go call the API makes has to be listed in
// docs/rbac-contract.yaml, and every entry there has to be used. The chart side is
// checked in helm-charts (scripts/ci/verify-api-rbac-contract.sh). Fake clientsets
// cannot see RBAC, so without this a new call only fails once it reaches a cluster.
//
// The scan finds chained calls such as Clientset.CoreV1().Namespaces().Patch(...).
// It does not see dynamic-client access or a resource client kept in a variable.

type rbacContract struct {
	Version int `json:"version"`
	Rules   []struct {
		APIGroup        string   `json:"apiGroup"`
		Resource        string   `json:"resource"`
		Scope           string   `json:"scope"`
		Verbs           []string `json:"verbs"`
		BestEffortVerbs []string `json:"bestEffortVerbs"`
	} `json:"rules"`
}

var typedClientGroups = map[string]string{
	"CoreV1":                  "",
	"AppsV1":                  "apps",
	"BatchV1":                 "batch",
	"NetworkingV1":            "networking.k8s.io",
	"RbacV1":                  "rbac.authorization.k8s.io",
	"StorageV1":               "storage.k8s.io",
	"PolicyV1":                "policy",
	"AuthorizationV1":         "authorization.k8s.io",
	"AutoscalingV2":           "autoscaling",
	"DiscoveryV1":             "discovery.k8s.io",
	"CoordinationV1":          "coordination.k8s.io",
	"SchedulingV1":            "scheduling.k8s.io",
	"AdmissionregistrationV1": "admissionregistration.k8s.io",
}

var typedClientVerbs = map[string]string{
	"Get": "get", "List": "list", "Create": "create", "Update": "update",
	"UpdateStatus": "update", "Patch": "patch", "Delete": "delete",
	"DeleteCollection": "deletecollection", "Watch": "watch",
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// scanTypedCalls returns "group|resource|verb" -> call sites.
func scanTypedCalls(t *testing.T, root string) map[string][]string {
	t.Helper()
	found := map[string][]string{}
	fset := token.NewFileSet()
	for _, sub := range []string{"cmd", "internal"} {
		_ = filepath.Walk(filepath.Join(root, sub), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			f, perr := parser.ParseFile(fset, p, nil, 0)
			if perr != nil {
				t.Fatalf("parse %s: %v", p, perr)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				verbSel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				resCall, ok := verbSel.X.(*ast.CallExpr) // X.Namespaces(...)
				if !ok {
					return true
				}
				resSel, ok := resCall.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				groupCall, ok := resSel.X.(*ast.CallExpr) // Clientset.CoreV1()
				if !ok {
					return true
				}
				groupSel, ok := groupCall.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				group, isTyped := typedClientGroups[groupSel.Sel.Name]
				if !isTyped {
					return true
				}
				resource := strings.ToLower(resSel.Sel.Name)
				verb, isVerb := typedClientVerbs[verbSel.Sel.Name]
				if verbSel.Sel.Name == "GetLogs" && resource == "pods" {
					resource, verb, isVerb = "pods/log", "get", true
				}
				if !isVerb {
					return true
				}
				rel, _ := filepath.Rel(root, p)
				key := group + "|" + resource + "|" + verb
				found[key] = append(found[key], fmt.Sprintf("%s:%d", rel, fset.Position(call.Pos()).Line))
				return true
			})
			return nil
		})
	}
	return found
}

func TestRBACContractMatchesTypedClientCalls(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "rbac-contract.yaml"))
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var contract rbacContract
	if err := yaml.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("parse contract: %v", err)
	}

	listed := map[string]bool{}
	for _, r := range contract.Rules {
		for _, v := range append(append([]string{}, r.Verbs...), r.BestEffortVerbs...) {
			listed[r.APIGroup+"|"+r.Resource+"|"+v] = true
		}
	}

	used := scanTypedCalls(t, root)
	if len(used) == 0 {
		t.Fatal("scanner found no typed client calls; the AST pattern is broken")
	}

	var missing []string
	for key, sites := range used {
		if !listed[key] {
			missing = append(missing, fmt.Sprintf("  %s  (%s)", key, strings.Join(sites, ", ")))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("code calls the Kubernetes API in ways docs/rbac-contract.yaml does not list "+
			"(group|resource|verb). Add them to the contract and grant them in helm-charts, or, if the "+
			"call tolerates Forbidden, list them under bestEffortVerbs:\n%s", strings.Join(missing, "\n"))
	}

	var stale []string
	for key := range listed {
		if _, ok := used[key]; !ok {
			stale = append(stale, "  "+key)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("docs/rbac-contract.yaml lists access the code no longer uses; remove it so the "+
			"chart does not over-grant:\n%s", strings.Join(stale, "\n"))
	}
}
