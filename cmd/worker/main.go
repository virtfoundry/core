package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/virtforge-cloud/virtforge/internal/config"
	"github.com/virtforge-cloud/virtforge/internal/infra/hypervisor"
	"github.com/virtforge-cloud/virtforge/internal/pkg/logger"
	platformk8s "github.com/virtforge-cloud/virtforge/internal/platform/k8s"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service"
	"go.uber.org/zap"
)

func main() {
	cfg := loadConfig()
	logger.Init(cfg.Logger.Level, cfg.Logger.Format != "json")
	log := logger.Get()
	log.Info("starting Nimbus worker")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, err := store.Open(cfg.Database)
	if err != nil {
		log.Fatal("open store", zap.Error(err))
	}
	defer repo.Close()

	inCluster := os.Getenv("KUBERNETES_SERVICE_HOST") != ""
	k8sMgr, err := platformk8s.NewManager(platformk8s.Options{
		Kubeconfig: cfg.KubeVirt.Kubeconfig,
		InCluster:  inCluster,
	})
	if err != nil {
		log.Fatal("k8s manager", zap.Error(err))
	}

	kvDriver, err := hypervisor.NewKubeVirtDriver(hypervisor.KubeVirtConfig{
		Kubeconfig: cfg.KubeVirt.Kubeconfig,
		Namespace:  cfg.KubeVirt.Namespace,
	})
	if err != nil {
		log.Fatal("kubevirt driver", zap.Error(err))
	}

	platformSvc := service.NewPlatformService(repo, k8sMgr, kvDriver, nil)

	jobTicker := time.NewTicker(3 * time.Second)
	reconcileTicker := time.NewTicker(15 * time.Second)
	defer jobTicker.Stop()
	defer reconcileTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-jobTicker.C:
				platformSvc.ProcessPendingJobs(ctx)
			case <-reconcileTicker.C:
				platformSvc.ReconcileAll(ctx)
				platformSvc.SyncAllVMStates(ctx)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	log.Info("worker shutdown complete")
}

func loadConfig() *config.Config {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
}
