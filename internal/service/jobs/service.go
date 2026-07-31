package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service/compute"
)

// Service processes the async job queue.
type Service struct {
	store   store.Repository
	compute *compute.Service
}

func New(st store.Repository, computeSvc *compute.Service) *Service {
	return &Service{store: st, compute: computeSvc}
}

func (s *Service) Enqueue(tenantID, jobType, payload string) *platform.AsyncJob {
	j := &platform.AsyncJob{
		ID: store.NewID(), TenantID: tenantID, Type: jobType, Payload: payload,
		Status: "pending", CreatedAt: store.Now(), UpdatedAt: store.Now(),
	}
	s.store.SaveJob(j)
	return j
}

func (s *Service) ProcessPending(ctx context.Context) {
	for _, j := range s.store.ListPendingJobs(20) {
		j.Status = "running"
		j.UpdatedAt = time.Now()
		s.store.SaveJob(j)

		var err error
		switch j.Type {
		case "deploy_vm":
			var in compute.DeployVMInput
			if e := json.Unmarshal([]byte(j.Payload), &in); e != nil {
				err = e
			} else {
				_, err = s.compute.DeployVM(ctx, j.TenantID, in)
			}
		case "reconcile":
			s.compute.ReconcileAll(ctx)
		default:
			err = fmt.Errorf("unknown job type: %s", j.Type)
		}
		if err != nil {
			j.Status = "failed"
			j.Error = err.Error()
		} else {
			j.Status = "succeeded"
		}
		j.UpdatedAt = time.Now()
		s.store.SaveJob(j)
	}
}
