package compute

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/virtfoundry/core/internal/platform/branding"

	logutil "github.com/virtfoundry/core/internal/platform/logs"
	"github.com/virtfoundry/core/internal/service/shared"
)

// VelasConfig holds external log explorer integration (Loki/Grafana-compatible URL template).
type VelasConfig struct {
	ExploreURLTemplate string // e.g. https://velas.example/explore?query={query}
}

var velasCfg VelasConfig

// SetVelasConfig configures the Velas log explorer link template.
func SetVelasConfig(cfg VelasConfig) {
	velasCfg = cfg
}

// StreamLogs tails virt-launcher output for a VM instance.
func (s *Service) StreamLogs(ctx context.Context, tenantID, vmName string, tailLines int64, follow bool) (io.ReadCloser, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	stream, err := s.k8s.StreamVMPodLogs(ctx, ns, vmName, tailLines, follow)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(logutil.SanitizeReader(stream)), nil
}

// LogExploreURL returns a Velas/Grafana explore link for the VM, if configured.
func (s *Service) LogExploreURL(tenantID, vmName string) string {
	tpl := velasCfg.ExploreURLTemplate
	if tpl == "" {
		return ""
	}
	query := fmt.Sprintf(`{%s="%s",%s="%s"}`, branding.LogLabelVM, vmName, branding.LogLabelTenant, tenantID)
	return strings.NewReplacer("{query}", query, "{vm}", vmName, "{tenant}", tenantID).Replace(tpl)
}
