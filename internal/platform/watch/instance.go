// Package watch provides CRD informers that publish platform events onto the
// realtime hub (UI /ws/events and gRPC WatchInstances).
package watch

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/virtfoundry/core/internal/pkg/logger"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	"github.com/virtfoundry/core/internal/service/shared"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

const (
	eventCreated = "vm.created"
	eventUpdated = "vm.updated"
	eventDeleted = "vm.deleted"

	defaultResync = 30 * time.Second
)

// VMEvent is the hub payload for Instance CR changes. It carries both desired
// power_state (spec) and observed state (status.phase via
// InstancePhaseToPlatformState) so Start/Stop UX under operatorReconcile is
// not status-only.
type VMEvent struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name"`
	State      string `json:"state,omitempty"`
	PowerState string `json:"power_state,omitempty"`
	IP         string `json:"ip,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
}

// TenantResolver maps an Instance namespace (or labels) to a platform tenant ID.
type TenantResolver func(obj *unstructured.Unstructured) string

// InstancePublisher maps Instance CR events onto a tenant-scoped hub.
type InstancePublisher struct {
	Hub      shared.EventBroadcaster
	Resolve  TenantResolver
	Log      *zap.Logger
}

// InstanceInformerEnabled reports whether the Instance SharedInformer should
// start. VF_INSTANCE_INFORMER=0/false disables; =1/true forces on. When unset,
// defaults to on for kubernetes store mode (operatorReconcile / CRD-first).
func InstanceInformerEnabled(kubernetesStore bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("VF_INSTANCE_INFORMER"))) {
	case "0", "false", "off", "no":
		return false
	case "1", "true", "on", "yes":
		return true
	default:
		return kubernetesStore
	}
}

// TenantResolverFromStore derives tenant_id the same way the kubernetes store
// maps Instances: namespace → Tenant.Namespace, else label virtfoundry.io/tenant
// (slug) → GetTenantBySlug.
func TenantResolverFromStore(repo store.Repository) TenantResolver {
	return func(obj *unstructured.Unstructured) string {
		if obj == nil || repo == nil {
			return ""
		}
		ns := obj.GetNamespace()
		for _, t := range repo.ListTenants() {
			if t != nil && t.Namespace != "" && t.Namespace == ns {
				return t.ID
			}
		}
		slug := ""
		if labels := obj.GetLabels(); labels != nil {
			slug = labels[mapping.LabelTenant]
		}
		if slug == "" && strings.HasPrefix(ns, branding.TenantNamespacePrefix) {
			slug = strings.TrimPrefix(ns, branding.TenantNamespacePrefix)
		}
		if slug == "" {
			return ""
		}
		if t, ok := repo.GetTenantBySlug(slug); ok && t != nil {
			return t.ID
		}
		return ""
	}
}

// PayloadFromInstance builds a full hub payload from an Instance unstructured.
func PayloadFromInstance(obj *unstructured.Unstructured, tenantID string) VMEvent {
	if obj == nil {
		return VMEvent{TenantID: tenantID}
	}
	ev := VMEvent{
		ID:       mapping.ResourceID(obj),
		Name:     obj.GetName(),
		TenantID: tenantID,
	}
	vm, err := mapping.InstanceFromUnstructured(obj, tenantID, nil)
	if err != nil || vm == nil {
		if phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase"); phase != "" {
			ev.State = mapping.InstancePhaseToPlatformState(phase)
		}
		if ps, _, _ := unstructured.NestedString(obj.Object, "spec", "powerState"); ps != "" {
			ev.PowerState = ps
		}
		if ip, _, _ := unstructured.NestedString(obj.Object, "status", "ip"); ip != "" {
			ev.IP = ip
		}
		return ev
	}
	ev.ID = vm.ID
	ev.Name = vm.Name
	ev.State = vm.State
	ev.PowerState = vm.PowerState
	ev.IP = vm.IP
	ev.TenantID = tenantID
	return ev
}

func (p *InstancePublisher) log() *zap.Logger {
	if p != nil && p.Log != nil {
		return p.Log
	}
	return logger.Get()
}

// OnAdd publishes vm.created.
func (p *InstancePublisher) OnAdd(obj interface{}) {
	u := asUnstructured(obj)
	if u == nil || p == nil || p.Hub == nil {
		return
	}
	tenantID := ""
	if p.Resolve != nil {
		tenantID = p.Resolve(u)
	}
	if tenantID == "" {
		p.log().Debug("instance informer: drop add without tenant",
			zap.String("name", u.GetName()), zap.String("namespace", u.GetNamespace()))
		return
	}
	p.Hub.BroadcastTenant(tenantID, eventCreated, PayloadFromInstance(u, tenantID))
}

// OnUpdate publishes vm.updated when desired/observed VM fields change.
func (p *InstancePublisher) OnUpdate(oldObj, newObj interface{}) {
	neu := asUnstructured(newObj)
	if neu == nil || p == nil || p.Hub == nil {
		return
	}
	tenantID := ""
	if p.Resolve != nil {
		tenantID = p.Resolve(neu)
	}
	if tenantID == "" {
		p.log().Debug("instance informer: drop update without tenant",
			zap.String("name", neu.GetName()), zap.String("namespace", neu.GetNamespace()))
		return
	}
	next := PayloadFromInstance(neu, tenantID)
	if old := asUnstructured(oldObj); old != nil {
		prevTenant := tenantID
		if p.Resolve != nil {
			if tid := p.Resolve(old); tid != "" {
				prevTenant = tid
			}
		}
		prev := PayloadFromInstance(old, prevTenant)
		if vmEventEqual(prev, next) {
			return
		}
	}
	p.Hub.BroadcastTenant(tenantID, eventUpdated, next)
}

// OnDelete publishes vm.deleted.
func (p *InstancePublisher) OnDelete(obj interface{}) {
	u := asUnstructured(obj)
	if u == nil || p == nil || p.Hub == nil {
		return
	}
	tenantID := ""
	if p.Resolve != nil {
		tenantID = p.Resolve(u)
	}
	if tenantID == "" {
		p.log().Debug("instance informer: drop delete without tenant",
			zap.String("name", u.GetName()), zap.String("namespace", u.GetNamespace()))
		return
	}
	ev := PayloadFromInstance(u, tenantID)
	// deleted payloads still carry id/name/tenant; state/power optional
	p.Hub.BroadcastTenant(tenantID, eventDeleted, ev)
}

func vmEventEqual(a, b VMEvent) bool {
	return a.ID == b.ID &&
		a.Name == b.Name &&
		a.State == b.State &&
		a.PowerState == b.PowerState &&
		a.IP == b.IP &&
		a.TenantID == b.TenantID
}

func asUnstructured(obj interface{}) *unstructured.Unstructured {
	if obj == nil {
		return nil
	}
	switch t := obj.(type) {
	case *unstructured.Unstructured:
		return t
	case unstructured.Unstructured:
		return &t
	case cache.DeletedFinalStateUnknown:
		return asUnstructured(t.Obj)
	default:
		return nil
	}
}

// Options configures StartInstanceInformer.
type Options struct {
	Dynamic    dynamic.Interface
	Hub        shared.EventBroadcaster
	Resolve    TenantResolver
	Resync     time.Duration
	Log        *zap.Logger
}

// StartInstanceInformer runs a cluster-scoped SharedInformer on Instance CRs
// and publishes full VM events to Hub. It blocks until ctx is cancelled.
// Callers typically invoke it in a goroutine after kube client bootstrap.
func StartInstanceInformer(ctx context.Context, opts Options) error {
	if opts.Dynamic == nil {
		return nil
	}
	if opts.Hub == nil {
		return nil
	}
	resync := opts.Resync
	if resync <= 0 {
		resync = defaultResync
	}
	log := opts.Log
	if log == nil {
		log = logger.Get()
	}
	pub := &InstancePublisher{Hub: opts.Hub, Resolve: opts.Resolve, Log: log}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(opts.Dynamic, resync)
	informer := factory.ForResource(mapping.InstanceGVR).Informer()
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    pub.OnAdd,
		UpdateFunc: pub.OnUpdate,
		DeleteFunc: pub.OnDelete,
	})
	if err != nil {
		return err
	}

	log.Info("instance informer starting",
		zap.String("gvr", mapping.InstanceGVR.String()),
		zap.Duration("resync", resync))

	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Warn("instance informer cache sync incomplete")
		return nil
	}
	log.Info("instance informer synced")
	<-ctx.Done()
	return ctx.Err()
}
