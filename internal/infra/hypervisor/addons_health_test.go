package hypervisor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func newCRDTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, kubernetes.Interface) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := &rest.Config{Host: srv.URL}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	return srv, cs
}

// crdWithLabel mirrors a PartialObjectMetadata item: metadata only, no
// spec/status — the probe requests the list with the meta.k8s.io transform.
func crdWithLabel(name, labelValue string) string {
	return `{
		"metadata": {
			"name": "` + name + `",
			"labels": {"app.kubernetes.io/name": "` + labelValue + `"}
		}
	}`
}

func crdBare(name string) string {
	return `{"metadata": {"name": "` + name + `"}}`
}

func TestAddonsHealthDiscoversFromCRDs(t *testing.T) {
	t.Parallel()
	srv, cs := newCRDTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
			if got := r.Header.Get("Accept"); !strings.Contains(got, "as=PartialObjectMetadataList") {
				t.Errorf("Accept=%q must request PartialObjectMetadataList (metadata only, no schemas)", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items": [
				` + crdWithLabel("virtualmachines.kubevirt.io", "kubevirt") + `,
				` + crdWithLabel("datavolumes.cdi.kubevirt.io", "cdi") + `,
				` + crdWithLabel("networkattachmentdefinitions.k8s.cni.cncf.io", "multus") + `,
				` + crdWithLabel("certificates.cert-manager.io", "cert-manager") + `,
				` + crdWithLabel("foos.example.com", "some-addon") + `,
				` + crdBare("istiooperators.install.istio.io") + `,
				` + crdBare("networkpolicies.networking.k8s.io") + `,
				` + crdBare("widgets.unknown.example.com") + `
			]}`))
		default:
			http.NotFound(w, r)
		}
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	got, err := d.AddonsHealth(context.Background())
	if err != nil {
		t.Fatalf("AddonsHealth: %v", err)
	}
	byName := map[string]string{}
	for _, a := range got.Addons {
		byName[a.Name] = a.Status
	}
	want := map[string]string{
		"kubevirt":     "ok",
		"cdi":          "ok",
		"multus":       "ok",
		"cert-manager": "ok",
		"some-addon":   "ok",
		"istio":        "ok",
	}
	for name, status := range want {
		if got := byName[name]; got != status {
			t.Fatalf("addon %q: got %q want %q", name, got, status)
		}
	}
	if _, present := byName["unknown"]; present {
		t.Fatal("CRDs whose group has no alias must be ignored")
	}
	if _, present := byName["networking"]; present {
		t.Fatal("kubernetes core networking group must not be mapped to an addon")
	}
	if got.CheckedAt.IsZero() {
		t.Fatal("checked_at must be set")
	}
}

func TestAliasFromGroup(t *testing.T) {
	t.Parallel()
	cases := []struct {
		group string
		want  string
	}{
		{"kubevirt.io", "kubevirt"},
		{"cdi.kubevirt.io", "cdi"},
		{"k8s.cni.cncf.io", "multus"},
		{"cert-manager.io", "cert-manager"},
		{"networking.istio.io", "istio"},
		{"security.istio.io", "istio"},
		{"monitoring.coreos.com", "prometheus"},
		{"tekton.dev", "tekton"},
		{"argoproj.io", "argocd"},
		{"networking.k8s.io", ""},
		{"", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.group, func(t *testing.T) {
			t.Parallel()
			if got := aliasFromGroup(c.group); got != c.want {
				t.Fatalf("aliasFromGroup(%q)=%q want %q", c.group, got, c.want)
			}
		})
	}
}

func TestGroupFromCRDName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		want string
	}{
		{"virtualmachines.kubevirt.io", "kubevirt.io"},
		{"datavolumes.cdi.kubevirt.io", "cdi.kubevirt.io"},
		{"networkpolicies.networking.k8s.io", "networking.k8s.io"},
		{"nodot", ""},
		{"trailingdot.", ""},
		{"", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := groupFromCRDName(c.name); got != c.want {
				t.Fatalf("groupFromCRDName(%q)=%q want %q", c.name, got, c.want)
			}
		})
	}
}

func TestResolveAddonName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		labels map[string]string
		group  string
		want   string
	}{
		{"name wins over group", map[string]string{"app.kubernetes.io/name": "x"}, "kubevirt.io", "x"},
		{"part-of fallback", map[string]string{"app.kubernetes.io/part-of": "y"}, "kubevirt.io", "y"},
		{"component fallback", map[string]string{"app.kubernetes.io/component": "z"}, "kubevirt.io", "z"},
		{"group alias fallback", nil, "kubevirt.io", "kubevirt"},
		{"nothing matches", nil, "random.example.com", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveAddonName(c.labels, c.group); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestAddonsHealthCRDListForbidden(t *testing.T) {
	t.Parallel()
	srv, cs := newCRDTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"kind":"Status","status":"Failure","code":403,"reason":"Forbidden"}`, http.StatusForbidden)
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	got, err := d.AddonsHealth(context.Background())
	if err != nil {
		t.Fatalf("AddonsHealth: %v", err)
	}
	if len(got.Addons) != 0 {
		t.Fatalf("forbidden CRD list must yield empty addons, got %+v", got.Addons)
	}
}

func TestAddonsHealthDedupMultipleCRDsSameName(t *testing.T) {
	t.Parallel()
	srv, cs := newCRDTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items": [
				` + crdWithLabel("virtualmachines.kubevirt.io", "kubevirt") + `,
				` + crdWithLabel("virtualmachineinstances.kubevirt.io", "kubevirt") + `
			]}`))
		default:
			http.NotFound(w, r)
		}
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	got, err := d.AddonsHealth(context.Background())
	if err != nil {
		t.Fatalf("AddonsHealth: %v", err)
	}
	count := 0
	for _, a := range got.Addons {
		if a.Name == "kubevirt" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("multiple CRDs with same app.kubernetes.io/name must dedup, got %d entries", count)
	}
}
