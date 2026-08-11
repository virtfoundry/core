package network

import (
	"context"

	"github.com/virtfoundry/core/internal/config"
	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/platform/store"
)

// BootstrapSharedNetwork registers platform public network from Helm config and seeds IP pool.
func (s *Service) BootstrapSharedNetwork(ctx context.Context, cfg config.PublicNetworkConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.NADNamespace == "" {
		cfg.NADNamespace = branding.SystemNamespace
	}
	if cfg.NADName == "" {
		cfg.NADName = "virtfoundry-public"
	}
	if cfg.BridgeName == "" {
		cfg.BridgeName = branding.PublicBridgeName
	}
	if err := branding.ValidateLinuxBridgeName(cfg.BridgeName); err != nil {
		return err
	}

	labels := platformk8s.NADLabels{
		branding.LabelNetworkRole: "shared",
	}
	if err := s.k8s.CreateSharedNetworkAttachment(ctx, platformk8s.SharedNetworkAttachment{
		Namespace:  cfg.NADNamespace,
		Name:       cfg.NADName,
		Bridge:     cfg.BridgeName,
		CIDR:       cfg.CIDR,
		Gateway:    cfg.Gateway,
		RangeStart: cfg.IPPoolStart,
		RangeEnd:   cfg.IPPoolEnd,
		Labels:     labels,
	}); err != nil {
		return err
	}

	net := &platform.Network{
		ID:           platform.SharedNetworkID,
		Name:         "public",
		NetworkType:  platform.NetworkTypeShared,
		CIDR:         cfg.CIDR,
		Gateway:      cfg.Gateway,
		NADNamespace: cfg.NADNamespace,
		NADName:      cfg.NADName,
		State:        "active",
		CreatedAt:    store.Now(),
	}
	s.store.SaveNetwork(net)
	return s.store.SeedIPPool(net.ID, cfg.IPPoolStart, cfg.IPPoolEnd)
}
