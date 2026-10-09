package hypervisor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func newMetricsServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, kubernetes.Interface) {
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

func TestClusterUsageAggregates(t *testing.T) {
	t.Parallel()
	srv, cs := newMetricsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apis/metrics.k8s.io/v1beta1/nodes" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"kind": "NodeMetricsList",
			"apiVersion": "metrics.k8s.io/v1beta1",
			"items": [
				{"usage": {"cpu": "250m", "memory": "512Mi"}, "timestamp": "2026-10-09T11:00:00Z"},
				{"usage": {"cpu": "1.5", "memory": "4Gi"}, "timestamp": "2026-10-09T11:00:00Z"},
				{"usage": {"cpu": "0", "memory": "0"}, "timestamp": "2026-10-09T11:00:00Z"}
			]
		}`))
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	u, err := d.ClusterUsage(context.Background())
	if err != nil {
		t.Fatalf("ClusterUsage: %v", err)
	}
	if u == nil {
		t.Fatal("expected non-nil usage")
	}
	if got, want := u.CPUUsageMilli, int64(250+1500+0); got != want {
		t.Fatalf("cpu_usage_millicores=%d want %d", got, want)
	}
	memWant := int64(512*1024*1024 + 4*1024*1024*1024)
	if u.MemoryUsage != memWant {
		t.Fatalf("memory_usage_bytes=%d want %d", u.MemoryUsage, memWant)
	}
	if u.WindowSeconds != int64(clusterUsageTTL/time.Second) {
		t.Fatalf("window_seconds=%d want %d", u.WindowSeconds, int64(clusterUsageTTL/time.Second))
	}
	if u.CollectedAt.IsZero() {
		t.Fatal("collected_at must be set")
	}
}

func TestClusterUsageMetricsServerAbsent(t *testing.T) {
	t.Parallel()
	srv, cs := newMetricsServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"kind":"Status","status":"Failure","code":404,"reason":"NotFound","message":"the server could not find the requested resource"}`, http.StatusNotFound)
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	u, err := d.ClusterUsage(context.Background())
	if err != nil {
		t.Fatalf("expected graceful nil on 404, got error: %v", err)
	}
	if u != nil {
		t.Fatalf("expected nil usage when metrics-server is absent, got %+v", u)
	}
}

func TestClusterUsageForbidden(t *testing.T) {
	t.Parallel()
	srv, cs := newMetricsServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"kind":"Status","status":"Failure","code":403,"reason":"Forbidden"}`, http.StatusForbidden)
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	u, err := d.ClusterUsage(context.Background())
	if err != nil {
		t.Fatalf("expected graceful nil on 403, got error: %v", err)
	}
	if u != nil {
		t.Fatalf("expected nil usage on Forbidden, got %+v", u)
	}
}

func TestClusterUsageTransientError(t *testing.T) {
	t.Parallel()
	srv, cs := newMetricsServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"kind":"Status","status":"Failure","code":503,"reason":"ServiceUnavailable"}`, http.StatusServiceUnavailable)
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	u, err := d.ClusterUsage(context.Background())
	if err != nil {
		t.Fatalf("expected graceful nil on 503, got error: %v", err)
	}
	if u != nil {
		t.Fatalf("expected nil usage on 503, got %+v", u)
	}
}

func TestClusterUsageParseError(t *testing.T) {
	t.Parallel()
	srv, cs := newMetricsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{ this is not json`))
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	if _, err := d.ClusterUsage(context.Background()); err == nil {
		t.Fatal("expected parse error to propagate")
	}
}

func TestClusterUsageCache(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv, cs := newMetricsServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items": [{"usage": {"cpu": "100m", "memory": "256Mi"}}]}`))
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	if _, err := d.ClusterUsage(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := d.ClusterUsage(context.Background()); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("cache did not absorb the second call: requests=%d want 1", got)
	}
}

func TestClusterUsageCacheTTLExpiry(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv, cs := newMetricsServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items": [{"usage": {"cpu": "100m", "memory": "256Mi"}}]}`))
	})
	_ = srv

	d := &KubeVirtDriver{k8sClient: cs}
	if _, err := d.ClusterUsage(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	d.usageCacheMu.Lock()
	d.usageCache.at = time.Now().Add(-2 * clusterUsageTTL)
	d.usageCacheMu.Unlock()
	if _, err := d.ClusterUsage(context.Background()); err != nil {
		t.Fatalf("third call: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("cache TTL did not expire: requests=%d want 2", got)
	}
}

func TestClusterUsageNilClient(t *testing.T) {
	t.Parallel()
	d := &KubeVirtDriver{}
	u, err := d.ClusterUsage(context.Background())
	if err != nil {
		t.Fatalf("nil client: %v", err)
	}
	if u != nil {
		t.Fatalf("nil client must yield nil usage, got %+v", u)
	}
}

func TestIsMetricsAbsent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"random", errors.New("some other failure"), false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := isMetricsAbsent(c.err); got != c.want {
				t.Fatalf("isMetricsAbsent(%v)=%v want %v", c.err, got, c.want)
			}
		})
	}
}
