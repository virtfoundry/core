package vks

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func testPod(ns, name string, phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Status:     corev1.PodStatus{Phase: phase},
	}
}

func testNode(name string, ready corev1.ConditionStatus, withCondition bool) *corev1.Node {
	n := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if withCondition {
		n.Status.Conditions = []corev1.NodeCondition{{Type: corev1.NodeReady, Status: ready}}
	}
	return n
}

func TestBuildSummaryFromClient(t *testing.T) {
	cs := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "empty"}},
		testPod("kube-system", "a", corev1.PodRunning),
		testPod("kube-system", "b", corev1.PodRunning),
		testPod("default", "c", corev1.PodPending),
		testPod("default", "d", corev1.PodFailed),
		testPod("default", "e", corev1.PodSucceeded),
		testNode("w2", corev1.ConditionFalse, true),
		testNode("w1", corev1.ConditionTrue, true),
		testNode("w3", "", false),
	)
	got, err := buildSummaryFromClient(context.Background(), cs)
	if err != nil {
		t.Fatal(err)
	}
	want := &Summary{
		Namespaces: []NamespaceSummary{
			{Name: "default", PodCount: 3},
			{Name: "empty", PodCount: 0},
			{Name: "kube-system", PodCount: 2},
		},
		Pods: PodTotals{Running: 2, Pending: 1, Failed: 1, Other: 1},
		Nodes: []GuestNode{
			{Name: "w1", Ready: true},
			{Name: "w2", Ready: false},
			{Name: "w3", Ready: false},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summary mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestBuildSummaryFromClientEmpty(t *testing.T) {
	got, err := buildSummaryFromClient(context.Background(), k8sfake.NewSimpleClientset())
	if err != nil {
		t.Fatal(err)
	}
	if got.Namespaces == nil || got.Nodes == nil || len(got.Namespaces) != 0 || len(got.Nodes) != 0 {
		t.Fatalf("want empty non-nil slices, got %+v", got)
	}
}
