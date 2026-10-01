package vfgrpc

import (
	"context"
	"time"

	vksv1alpha1 "github.com/virtfoundry/core/api/gen/virtfoundry/vks/v1alpha1"
	"github.com/virtfoundry/core/internal/auth"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	vkssvc "github.com/virtfoundry/core/internal/service/vks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClusterBackend is the domain facade used by ClusterService.
type ClusterBackend interface {
	List(ctx context.Context, tenantID string) ([]vkssvc.Cluster, error)
	Get(ctx context.Context, tenantID, name string) (*vkssvc.Cluster, error)
	Create(ctx context.Context, tenantID string, in vkssvc.CreateInput) (*vkssvc.Cluster, error)
	Delete(ctx context.Context, tenantID, name string) error
	GetKubeconfig(ctx context.Context, tenantID, name string) ([]byte, error)
}

// ClusterServer implements virtfoundry.vks.v1alpha1.ClusterService.
type ClusterServer struct {
	vksv1alpha1.UnimplementedClusterServiceServer
	Backend ClusterBackend
}

var _ vksv1alpha1.ClusterServiceServer = (*ClusterServer)(nil)

func (s *ClusterServer) ListClusters(ctx context.Context, req *vksv1alpha1.ListClustersRequest) (*vksv1alpha1.ListClustersResponse, error) {
	if err := requirePerm(ctx, auth.PermVKSRead); err != nil {
		return nil, err
	}
	tid := TenantIDFromContext(ctx)
	if tid == "" {
		return nil, status.Error(codes.Unauthenticated, "tenant required")
	}
	if s.Backend == nil {
		return nil, status.Error(codes.Internal, "cluster backend not configured")
	}
	list, err := s.Backend.List(ctx, tid)
	if err != nil {
		return nil, mapSvcErr(err)
	}
	capN := softListCap
	if req != nil && req.PageSize > 0 && int(req.PageSize) < capN {
		capN = int(req.PageSize)
	}
	out := make([]*vksv1alpha1.Cluster, 0, min(len(list), capN))
	for i := range list {
		if i >= capN {
			break
		}
		out = append(out, toProtoCluster(&list[i]))
	}
	return &vksv1alpha1.ListClustersResponse{Clusters: out}, nil
}

func (s *ClusterServer) GetCluster(ctx context.Context, req *vksv1alpha1.GetClusterRequest) (*vksv1alpha1.GetClusterResponse, error) {
	if err := requirePerm(ctx, auth.PermVKSRead); err != nil {
		return nil, err
	}
	tid := TenantIDFromContext(ctx)
	if tid == "" || req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name and tenant required")
	}
	c, err := s.Backend.Get(ctx, tid, req.Name)
	if err != nil {
		return nil, mapSvcErr(err)
	}
	return &vksv1alpha1.GetClusterResponse{Cluster: toProtoCluster(c)}, nil
}

func (s *ClusterServer) CreateCluster(ctx context.Context, req *vksv1alpha1.CreateClusterRequest) (*vksv1alpha1.CreateClusterResponse, error) {
	if err := requirePerm(ctx, auth.PermVKSWrite); err != nil {
		return nil, err
	}
	tid := TenantIDFromContext(ctx)
	if tid == "" || req == nil {
		return nil, status.Error(codes.InvalidArgument, "request and tenant required")
	}
	in := fromCreateRequest(req)
	c, err := s.Backend.Create(ctx, tid, in)
	if err != nil {
		return nil, mapSvcErr(err)
	}
	return &vksv1alpha1.CreateClusterResponse{Cluster: toProtoCluster(c)}, nil
}

func (s *ClusterServer) DeleteCluster(ctx context.Context, req *vksv1alpha1.DeleteClusterRequest) (*vksv1alpha1.DeleteClusterResponse, error) {
	if err := requirePerm(ctx, auth.PermVKSWrite); err != nil {
		return nil, err
	}
	tid := TenantIDFromContext(ctx)
	if tid == "" || req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name and tenant required")
	}
	if err := s.Backend.Delete(ctx, tid, req.Name); err != nil {
		return nil, mapSvcErr(err)
	}
	return &vksv1alpha1.DeleteClusterResponse{}, nil
}

func (s *ClusterServer) GetKubeconfig(ctx context.Context, req *vksv1alpha1.GetKubeconfigRequest) (*vksv1alpha1.GetKubeconfigResponse, error) {
	if err := requirePerm(ctx, auth.PermVKSKubeconfig); err != nil {
		return nil, err
	}
	tid := TenantIDFromContext(ctx)
	if tid == "" || req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name and tenant required")
	}
	raw, err := s.Backend.GetKubeconfig(ctx, tid, req.Name)
	if err != nil {
		return nil, mapSvcErr(err)
	}
	return &vksv1alpha1.GetKubeconfigResponse{Kubeconfig: raw}, nil
}

// WatchClusters sends an initial MODIFIED snapshot, then re-lists every 2s and emits diffs.
func (s *ClusterServer) WatchClusters(_ *vksv1alpha1.WatchClustersRequest, stream vksv1alpha1.ClusterService_WatchClustersServer) error {
	if err := requirePerm(stream.Context(), auth.PermVKSRead); err != nil {
		return err
	}
	tid := TenantIDFromContext(stream.Context())
	if tid == "" {
		return status.Error(codes.Unauthenticated, "tenant required")
	}
	if s.Backend == nil {
		return status.Error(codes.Internal, "cluster backend not configured")
	}

	prev := map[string]vkssvc.Cluster{}
	emit := func(event string, c *vkssvc.Cluster) error {
		return stream.Send(&vksv1alpha1.WatchClustersResponse{
			Cluster:   toProtoCluster(c),
			EventType: event,
		})
	}

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		list, err := s.Backend.List(stream.Context(), tid)
		if err != nil {
			return mapSvcErr(err)
		}
		cur := map[string]vkssvc.Cluster{}
		for i := range list {
			c := list[i]
			cur[c.Name] = c
			if old, ok := prev[c.Name]; !ok {
				if err := emit("ADDED", &c); err != nil {
					return err
				}
			} else if old.Phase != c.Phase || old.ReadyWorkers != c.ReadyWorkers || old.ControlPlaneEndpoint != c.ControlPlaneEndpoint {
				if err := emit("MODIFIED", &c); err != nil {
					return err
				}
			}
		}
		for name, old := range prev {
			if _, ok := cur[name]; !ok {
				if err := emit("DELETED", &old); err != nil {
					return err
				}
			}
		}
		prev = cur

		select {
		case <-stream.Context().Done():
			return nil
		case <-tick.C:
		}
	}
}

func requirePerm(ctx context.Context, perm string) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return status.Error(codes.Unauthenticated, "missing actor")
	}
	if actor.Role == platform.RoleRoot {
		return nil
	}
	if auth.HasPermission(actor.Permissions, perm) {
		return nil
	}
	return status.Errorf(codes.PermissionDenied, "%s required", perm)
}

func mapSvcErr(err error) error {
	if err == nil {
		return nil
	}
	var ia *iaerrors.IaaSError
	if e, ok := err.(*iaerrors.IaaSError); ok {
		ia = e
		switch ia.HTTPStatus() {
		case 400:
			return status.Error(codes.InvalidArgument, ia.Message)
		case 404:
			return status.Error(codes.NotFound, ia.Message)
		case 409:
			return status.Error(codes.FailedPrecondition, ia.Message+": "+ia.Detail)
		case 403:
			return status.Error(codes.PermissionDenied, ia.Message)
		default:
			return status.Error(codes.Internal, ia.Message)
		}
	}
	return status.Errorf(codes.Internal, "%v", err)
}

func toProtoCluster(c *vkssvc.Cluster) *vksv1alpha1.Cluster {
	if c == nil {
		return nil
	}
	out := &vksv1alpha1.Cluster{
		Name:                 c.Name,
		TenantId:             c.TenantID,
		Namespace:            c.Namespace,
		KubernetesVersion:    c.KubernetesVersion,
		Phase:                c.Phase,
		ControlPlaneEndpoint: c.ControlPlaneEndpoint,
		ReadyWorkers:         c.ReadyWorkers,
		KubeconfigSecretRef:  c.KubeconfigSecretRef,
		ControlPlane: &vksv1alpha1.ControlPlaneSpec{
			ServiceType: c.ControlPlane.ServiceType,
			Address:     c.ControlPlane.Address,
			Port:        c.ControlPlane.Port,
		},
		Workers: &vksv1alpha1.WorkersSpec{
			Count:       c.Workers.Count,
			TemplateRef: &vksv1alpha1.LocalObjectRef{Name: c.Workers.TemplateRef.Name},
			OfferingRef: &vksv1alpha1.LocalObjectRef{Name: c.Workers.OfferingRef.Name},
			NetworkRef:  &vksv1alpha1.LocalObjectRef{Name: c.Workers.NetworkRef.Name},
		},
	}
	for _, r := range c.Workers.SSHKeyRefs {
		out.Workers.SshKeyRefs = append(out.Workers.SshKeyRefs, &vksv1alpha1.LocalObjectRef{Name: r.Name})
	}
	return out
}

func fromCreateRequest(req *vksv1alpha1.CreateClusterRequest) vkssvc.CreateInput {
	in := vkssvc.CreateInput{
		Name:              req.Name,
		KubernetesVersion: req.KubernetesVersion,
	}
	if req.ControlPlane != nil {
		in.ControlPlane = vkssvc.ControlPlaneSpec{
			ServiceType: req.ControlPlane.ServiceType,
			Address:     req.ControlPlane.Address,
			Port:        req.ControlPlane.Port,
		}
	}
	if req.Workers != nil {
		in.Workers.Count = req.Workers.Count
		if req.Workers.TemplateRef != nil {
			in.Workers.TemplateRef.Name = req.Workers.TemplateRef.Name
		}
		if req.Workers.OfferingRef != nil {
			in.Workers.OfferingRef.Name = req.Workers.OfferingRef.Name
		}
		if req.Workers.NetworkRef != nil {
			in.Workers.NetworkRef.Name = req.Workers.NetworkRef.Name
		}
		for _, r := range req.Workers.SshKeyRefs {
			if r != nil && r.Name != "" {
				in.Workers.SSHKeyRefs = append(in.Workers.SSHKeyRefs, vkssvc.LocalObjectRef{Name: r.Name})
			}
		}
	}
	return in
}
