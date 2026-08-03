package service

import (
	"github.com/virtfoundry/core/internal/service/compute"
)

// Input types re-exported for handlers and migrate CLI.
type (
	PlatformDeployVMInput = compute.DeployVMInput
	UpdateVMInput         = compute.UpdateVMInput
	CreateVMTemplateInput = compute.CreateVMTemplateInput
)
