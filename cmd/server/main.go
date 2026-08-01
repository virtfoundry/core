package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/virtforge-cloud/virtforge/internal/api/handler"
	"github.com/virtforge-cloud/virtforge/internal/api/middleware"
	"github.com/virtforge-cloud/virtforge/internal/api/ws"
	"github.com/virtforge-cloud/virtforge/internal/auth"
	"github.com/virtforge-cloud/virtforge/internal/config"
	"github.com/virtforge-cloud/virtforge/internal/platform/branding"
	"github.com/virtforge-cloud/virtforge/internal/infra/hypervisor"
	"github.com/virtforge-cloud/virtforge/internal/pkg/logger"
	platformk8s "github.com/virtforge-cloud/virtforge/internal/platform/k8s"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service"
	"github.com/virtforge-cloud/virtforge/internal/service/compute"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func main() {
	cfg := loadConfig()
	logger.Init(cfg.Logger.Level, cfg.Logger.Format != "json")
	log := logger.Get()
	log.Info("starting VirtForge Cloud", zap.Int("port", cfg.Server.Port))

	hub := ws.NewHub()

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

	repo, err := store.Open(cfg.Database)
	if err != nil {
		log.Fatal("open store", zap.Error(err))
	}
	defer repo.Close()

	jwtSecret := cfg.Security.JWTSecret
	if v := os.Getenv("JWT_SECRET"); v != "" {
		jwtSecret = v
	}
	authSvc := auth.NewService(jwtSecret, cfg.Security.JWTExpire)
	platformSvc := service.NewPlatformService(repo, k8sMgr, kvDriver, hub)
	if cfg.Observability.VelasExploreURL != "" {
		compute.SetVelasConfig(compute.VelasConfig{ExploreURLTemplate: cfg.Observability.VelasExploreURL})
	}
	if v := os.Getenv("VELAS_EXPLORE_URL"); v != "" {
		compute.SetVelasConfig(compute.VelasConfig{ExploreURLTemplate: v})
	}

	if !repo.HasRootUser() {
		rootPass := os.Getenv("ROOT_PASSWORD")
		if rootPass == "" {
			rootPass = branding.DefaultRootPassword
		}
		if _, err := platformSvc.BootstrapRoot("root", rootPass); err != nil {
			log.Fatal("bootstrap root", zap.Error(err))
		}
		log.Info("root user bootstrapped", zap.String("username", "root"))
	}

	if tenant, err := platformSvc.BootstrapRootDefaultTenant(context.Background()); err != nil {
		log.Fatal("bootstrap default tenant", zap.Error(err))
	} else {
		log.Info("default tenant ready", zap.String("slug", tenant.Slug), zap.String("id", tenant.ID))
	}

	if err := platformSvc.BootstrapNetworking(context.Background(), cfg.Networking); err != nil {
		log.Fatal("bootstrap networking", zap.Error(err))
	}
	if cfg.Networking.Public.Enabled {
		log.Info("public network enabled",
			zap.String("cidr", cfg.Networking.Public.CIDR),
			zap.String("pool", cfg.Networking.Public.IPPoolStart+"-"+cfg.Networking.Public.IPPoolEnd))
	}

	router := mux.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.CORS)

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"virtforge-iaas","hypervisor":"kubevirt"}`))
	}).Methods("GET")

	router.HandleFunc("/ws/events", func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := hub.Register(conn)
		go client.WritePump()
		client.ReadPump()
	})

	consoleHandler := handler.NewConsoleHandler(kvDriver)
	router.HandleFunc("/ws/console", consoleHandler.VNCConsole)

	platformHandler := handler.NewPlatformHandler(authSvc, repo, platformSvc)

	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/auth/login", platformHandler.Login).Methods("POST")

	protected := v1.NewRoute().Subrouter()
	protected.Use(middleware.JWTAuth(authSvc))
	protected.Use(middleware.AuditRootImpersonation(repo))
	protected.HandleFunc("/auth/me", platformHandler.Me).Methods("GET")
	protected.HandleFunc("/vpcs", platformHandler.ListVPCs).Methods("GET")
	protected.HandleFunc("/vpcs/cidr-plan", platformHandler.VPCCIDRPlan).Methods("GET")
	protected.HandleFunc("/vpcs", platformHandler.CreateVPC).Methods("POST")
	protected.HandleFunc("/vpcs/{id}", platformHandler.UpdateVPC).Methods("PATCH")
	protected.HandleFunc("/vpcs/{id}", platformHandler.DeleteVPC).Methods("DELETE")
	protected.HandleFunc("/security-groups", platformHandler.ListSecurityGroups).Methods("GET")
	protected.HandleFunc("/security-groups", platformHandler.CreateSecurityGroup).Methods("POST")
	protected.HandleFunc("/security-groups/{id}", platformHandler.UpdateSecurityGroup).Methods("PATCH")
	protected.HandleFunc("/security-groups/{id}", platformHandler.DeleteSecurityGroup).Methods("DELETE")
	protected.HandleFunc("/networks", platformHandler.ListNetworks).Methods("GET")
	protected.HandleFunc("/networks/cidr-plan", platformHandler.NetworkCIDRPlan).Methods("GET")
	protected.HandleFunc("/networks", platformHandler.CreateNetwork).Methods("POST")
	protected.HandleFunc("/networks/{id}", platformHandler.UpdateNetwork).Methods("PATCH")
	protected.HandleFunc("/networks/{id}", platformHandler.DeleteNetwork).Methods("DELETE")
	protected.HandleFunc("/volumes", platformHandler.ListVolumes).Methods("GET")
	protected.HandleFunc("/volumes", platformHandler.CreateVolume).Methods("POST")
	protected.HandleFunc("/snapshots", platformHandler.ListSnapshots).Methods("GET")
	protected.HandleFunc("/snapshots", platformHandler.CreateSnapshot).Methods("POST")
	protected.HandleFunc("/vm-snapshots", platformHandler.ListVMSnapshots).Methods("GET")
	protected.HandleFunc("/vm-snapshots", platformHandler.CreateVMSnapshot).Methods("POST")
	protected.HandleFunc("/vm-snapshots/delete", platformHandler.DeleteVMSnapshot).Methods("POST")
	protected.HandleFunc("/vm-snapshots/restore", platformHandler.RestoreVMSnapshot).Methods("POST")
	protected.HandleFunc("/service-offerings", platformHandler.ListServiceOfferings).Methods("GET")
	protected.HandleFunc("/vm-templates", platformHandler.ListVMTemplates).Methods("GET")
	protected.HandleFunc("/vms", platformHandler.ListVMs).Methods("GET")
	protected.HandleFunc("/vms", platformHandler.DeployVM).Methods("POST")
	protected.HandleFunc("/vms/{name}/logs", platformHandler.GetVMLogs).Methods("GET")
	protected.HandleFunc("/vms/{name}", platformHandler.GetVM).Methods("GET")
	protected.HandleFunc("/vms/{name}", platformHandler.UpdateVM).Methods("PATCH")
	protected.HandleFunc("/vms/start", platformHandler.StartVM).Methods("POST")
	protected.HandleFunc("/vms/stop", platformHandler.StopVM).Methods("POST")
	protected.HandleFunc("/vms/delete", platformHandler.DeleteVM).Methods("POST")
	protected.HandleFunc("/ssh-keys", platformHandler.ListSSHKeys).Methods("GET")
	protected.HandleFunc("/ssh-keys", platformHandler.CreateSSHKey).Methods("POST")
	protected.HandleFunc("/ssh-keys/register", platformHandler.RegisterSSHKey).Methods("POST")
	protected.HandleFunc("/ssh-keys/{id}", platformHandler.DeleteSSHKey).Methods("DELETE")
	protected.HandleFunc("/vms/{name}/ssh", platformHandler.GetVMSSH).Methods("GET")
	protected.HandleFunc("/vms/{name}/ssh", platformHandler.ExposeVMSSH).Methods("POST")

	rootOnly := protected.NewRoute().Subrouter()
	rootOnly.Use(middleware.RequireRoot)
	rootOnly.HandleFunc("/tenants", platformHandler.ListTenants).Methods("GET")
	rootOnly.HandleFunc("/tenants", platformHandler.CreateTenant).Methods("POST")

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			platformSvc.SyncAllVMStates(context.Background())
		}
	}()

	go func() {
		log.Info("server listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Info("shutdown complete")
}

func loadConfig() *config.Config {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
}
