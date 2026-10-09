package hypervisor

import (
	"context"
	"testing"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newNode(name string, ready bool, cpuCap, cpuAlloc, memCap, memAlloc string, kubelet, osImage, arch string) *k8sv1.Node {
	node := &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: k8sv1.NodeStatus{
			Capacity: k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse(cpuCap),
				k8sv1.ResourceMemory: resource.MustParse(memCap),
			},
			Allocatable: k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse(cpuAlloc),
				k8sv1.ResourceMemory: resource.MustParse(memAlloc),
			},
			NodeInfo: k8sv1.NodeSystemInfo{
				KubeletVersion: kubelet,
				OSImage:        osImage,
				Architecture:   arch,
			},
		},
	}
	if ready {
		node.Status.Conditions = []k8sv1.NodeCondition{
			{Type: k8sv1.NodeReady, Status: k8sv1.ConditionTrue},
		}
	} else {
		node.Status.Conditions = []k8sv1.NodeCondition{
			{Type: k8sv1.NodeReady, Status: k8sv1.ConditionFalse},
		}
	}
	return node
}

func TestClusterMetricsAggregates(t *testing.T) {
	t.Parallel()
	cs := fake.NewSimpleClientset(
		newNode("a", true, "4", "3500m", "16Gi", "14Gi", "v1.30.0", "Ubuntu 22.04", "amd64"),
		newNode("b", true, "8", "7", "32Gi", "30Gi", "v1.30.0", "Ubuntu 22.04", "amd64"),
		newNode("c", false, "4", "3500m", "16Gi", "14Gi", "v1.29.5", "Debian 12", "arm64"),
	)
	d := &KubeVirtDriver{k8sClient: cs}
	m, err := d.ClusterMetrics(context.Background())
	if err != nil {
		t.Fatalf("ClusterMetrics: %v", err)
	}
	if m.Nodes != 3 {
		t.Fatalf("nodes=%d want 3", m.Nodes)
	}
	if m.NodesReady != 2 {
		t.Fatalf("nodes_ready=%d want 2", m.NodesReady)
	}
	if m.CPUCapacityMilli != 16000 {
		t.Fatalf("cpu_capacity_millicores=%d want 16000", m.CPUCapacityMilli)
	}
	wantAlloc := int64(14000) // 3.5 + 7 + 3.5 cores, in millicores
	if m.CPUAllocatableMilli != wantAlloc {
		t.Fatalf("cpu_allocatable_millicores=%d want %d", m.CPUAllocatableMilli, wantAlloc)
	}
	if m.MemoryCapacity != 64*1024*1024*1024 {
		t.Fatalf("memory_capacity=%d want 64Gi", m.MemoryCapacity)
	}
	if m.MemoryAllocatable != 58*1024*1024*1024 {
		t.Fatalf("memory_allocatable=%d want 58Gi", m.MemoryAllocatable)
	}
	if len(m.KubeletVersions) != 2 {
		t.Fatalf("kubelet_versions=%v want 2 distinct", m.KubeletVersions)
	}
	if len(m.OSArchitectures) != 2 {
		t.Fatalf("os_architectures=%v want 2 distinct", m.OSArchitectures)
	}
	if len(m.OSImages) != 2 {
		t.Fatalf("os_images=%v want 2 distinct", m.OSImages)
	}
	if m.CollectedAt.IsZero() {
		t.Fatal("collected_at must be set")
	}
}

func TestClusterMetricsEmpty(t *testing.T) {
	t.Parallel()
	cs := fake.NewSimpleClientset()
	d := &KubeVirtDriver{k8sClient: cs}
	m, err := d.ClusterMetrics(context.Background())
	if err != nil {
		t.Fatalf("ClusterMetrics: %v", err)
	}
	if m.Nodes != 0 || m.NodesReady != 0 || m.CPUAllocatableMilli != 0 {
		t.Fatalf("empty cluster: %+v", m)
	}
	if m.KubeletVersions == nil || m.OSImages == nil || m.OSArchitectures == nil {
		t.Fatal("slice fields must be non-nil (empty JSON arrays)")
	}
}

func TestIsNodeReady(t *testing.T) {
	t.Parallel()
	ready := newNode("a", true, "1", "1", "1Gi", "1Gi", "v1", "x", "amd64")
	if !isNodeReady(ready) {
		t.Fatal("expected ready")
	}
	notReady := newNode("b", false, "1", "1", "1Gi", "1Gi", "v1", "x", "amd64")
	if isNodeReady(notReady) {
		t.Fatal("expected not ready")
	}
	unknown := &k8sv1.Node{}
	if isNodeReady(unknown) {
		t.Fatal("node without conditions should not be ready")
	}
}