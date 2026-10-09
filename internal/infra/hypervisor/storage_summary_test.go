package hypervisor

import (
	"context"
	"testing"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newPVC(name, size string, phase k8sv1.PersistentVolumeClaimPhase) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			Resources: k8sv1.VolumeResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
		Status: k8sv1.PersistentVolumeClaimStatus{Phase: phase},
	}
}

func TestStorageSummaryAggregates(t *testing.T) {
	t.Parallel()
	cs := fake.NewSimpleClientset(
		newPVC("vol-1", "10Gi", k8sv1.ClaimBound),
		newPVC("vol-2", "20Gi", k8sv1.ClaimBound),
		newPVC("vol-3", "5Gi", k8sv1.ClaimPending),
		newPVC("vol-4", "1Gi", k8sv1.ClaimLost),
	)
	d := &KubeVirtDriver{k8sClient: cs, namespace: "default"}
	s, err := d.StorageSummary(context.Background())
	if err != nil {
		t.Fatalf("StorageSummary: %v", err)
	}
	if s.Count != 4 {
		t.Fatalf("count=%d want 4", s.Count)
	}
	wantTotal := int64((10 + 20 + 5 + 1) * 1024 * 1024 * 1024)
	if s.TotalBytes != wantTotal {
		t.Fatalf("total_bytes=%d want %d", s.TotalBytes, wantTotal)
	}
	wantUsed := int64((10 + 20) * 1024 * 1024 * 1024)
	if s.UsedBytes != wantUsed {
		t.Fatalf("used_bytes=%d want %d", s.UsedBytes, wantUsed)
	}
	if s.AvailableBytes != s.TotalBytes-s.UsedBytes {
		t.Fatalf("available_bytes=%d want total-used=%d", s.AvailableBytes, s.TotalBytes-s.UsedBytes)
	}
}

func TestStorageSummaryEmpty(t *testing.T) {
	t.Parallel()
	cs := fake.NewSimpleClientset()
	d := &KubeVirtDriver{k8sClient: cs, namespace: "default"}
	s, err := d.StorageSummary(context.Background())
	if err != nil {
		t.Fatalf("StorageSummary: %v", err)
	}
	if s.Count != 0 || s.TotalBytes != 0 || s.UsedBytes != 0 || s.AvailableBytes != 0 {
		t.Fatalf("expected zero-value summary, got %+v", s)
	}
}

func TestStorageSummaryNilClient(t *testing.T) {
	t.Parallel()
	d := &KubeVirtDriver{}
	s, err := d.StorageSummary(context.Background())
	if err != nil {
		t.Fatalf("nil client: %v", err)
	}
	if s != nil {
		t.Fatalf("nil client must yield nil summary, got %+v", s)
	}
}

func TestStorageSummarySkipsPVCsWithoutStorageRequest(t *testing.T) {
	t.Parallel()
	cs := fake.NewSimpleClientset(
		newPVC("vol-1", "10Gi", k8sv1.ClaimBound),
		&k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "vol-bare", Namespace: "default"},
			Spec: k8sv1.PersistentVolumeClaimSpec{
				Resources: k8sv1.VolumeResourceRequirements{Requests: k8sv1.ResourceList{}},
			},
			Status: k8sv1.PersistentVolumeClaimStatus{Phase: k8sv1.ClaimBound},
		},
	)
	d := &KubeVirtDriver{k8sClient: cs, namespace: "default"}
	s, err := d.StorageSummary(context.Background())
	if err != nil {
		t.Fatalf("StorageSummary: %v", err)
	}
	if s.Count != 2 {
		t.Fatalf("count=%d want 2 (PVCs without storage request must still be counted)", s.Count)
	}
	if s.TotalBytes != 10*1024*1024*1024 {
		t.Fatalf("total_bytes=%d want 10Gi", s.TotalBytes)
	}
}
