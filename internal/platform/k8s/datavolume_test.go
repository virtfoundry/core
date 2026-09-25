package k8s

import (
	"context"
	"errors"
	"testing"

	"github.com/virtfoundry/core/internal/platform/importurl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestCreateHTTPImportDataVolumeRefusesUnsafeURL(t *testing.T) {
	// Dynamic is nil on purpose: the URL must be refused before any API call.
	m := &Manager{}

	for _, isoURL := range []string{
		"https://169.254.169.254/latest/meta-data/",
		"https://kubernetes.default.svc/api/v1/secrets",
		"https://10.0.0.5/win2022.iso",
		"https://127.0.0.1/win2022.iso",
		"https://nas.local/win2022.iso",
		"http://iso.example.com/win2022.iso",
		"",
	} {
		err := m.CreateHTTPImportDataVolume(context.Background(), "vf-acme", "win-iso", isoURL, "local-path", 8)
		if err == nil {
			t.Fatalf("CreateHTTPImportDataVolume(%q) = nil, want rejection", isoURL)
		}
		if !errors.Is(err, importurl.ErrRejected) {
			t.Fatalf("CreateHTTPImportDataVolume(%q) error %v does not wrap ErrRejected", isoURL, err)
		}
	}
}

func TestCreateHTTPImportDataVolumeCreatesDataVolumeForPublicURL(t *testing.T) {
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{dataVolumeGVR: "DataVolumeList"})
	m := &Manager{Dynamic: client}
	ctx := context.Background()

	if err := m.CreateHTTPImportDataVolume(ctx, "vf-acme", "win-iso",
		"https://iso.example.com/win2022.iso", "local-path", 8); err != nil {
		t.Fatalf("CreateHTTPImportDataVolume = %v, want nil", err)
	}

	obj, err := client.Resource(dataVolumeGVR).Namespace("vf-acme").Get(ctx, "win-iso", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get datavolume: %v", err)
	}
	got, _, err := unstructured.NestedString(obj.Object, "spec", "source", "http", "url")
	if err != nil {
		t.Fatalf("read spec.source.http.url: %v", err)
	}
	if got != "https://iso.example.com/win2022.iso" {
		t.Fatalf("spec.source.http.url = %q", got)
	}
}
