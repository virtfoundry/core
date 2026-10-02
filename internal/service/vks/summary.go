package vks

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	summaryTimeout = 8 * time.Second
	summaryQPS     = 20
	summaryBurst   = 40
)

// GetSummary connects to the guest cluster via its admin kubeconfig and
// returns namespaces, pod totals and node readiness.
func (s *Service) GetSummary(ctx context.Context, tenantID, name string) (*Summary, error) {
	raw, err := s.GetKubeconfig(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("parse guest kubeconfig: %w", err)
	}
	cfg.QPS = summaryQPS
	cfg.Burst = summaryBurst
	cfg.Timeout = summaryTimeout
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("build guest client: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, summaryTimeout)
	defer cancel()
	return buildSummaryFromClient(ctx, cs)
}

// buildSummaryFromClient lists namespaces, pods (all namespaces) and nodes and aggregates them.
func buildSummaryFromClient(ctx context.Context, cs kubernetes.Interface) (*Summary, error) {
	nsList, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list guest namespaces: %w", err)
	}
	podList, err := cs.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list guest pods: %w", err)
	}
	nodeList, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list guest nodes: %w", err)
	}

	podsByNS := make(map[string]int32, len(nsList.Items))
	var totals PodTotals
	for i := range podList.Items {
		p := &podList.Items[i]
		podsByNS[p.Namespace]++
		switch p.Status.Phase {
		case corev1.PodRunning:
			totals.Running++
		case corev1.PodPending:
			totals.Pending++
		case corev1.PodFailed:
			totals.Failed++
		default:
			totals.Other++
		}
	}

	out := &Summary{
		Namespaces: make([]NamespaceSummary, 0, len(nsList.Items)),
		PodTotals:  totals,
		GuestNodes: make([]GuestNode, 0, len(nodeList.Items)),
	}
	for i := range nsList.Items {
		n := nsList.Items[i].Name
		out.Namespaces = append(out.Namespaces, NamespaceSummary{Name: n, PodCount: podsByNS[n]})
	}
	sort.Slice(out.Namespaces, func(i, j int) bool { return out.Namespaces[i].Name < out.Namespaces[j].Name })

	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		out.GuestNodes = append(out.GuestNodes, GuestNode{Name: node.Name, Ready: nodeReady(node)})
	}
	sort.Slice(out.GuestNodes, func(i, j int) bool { return out.GuestNodes[i].Name < out.GuestNodes[j].Name })
	return out, nil
}

func nodeReady(n *corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
