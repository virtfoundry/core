package compute

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
)

func TestCanDeployViaOperator(t *testing.T) {
	s := &Service{operatorReconcile: true}
	linuxTmpl := &platform.VMTemplate{Name: "cirros", SourceType: "container"}
	isoTmpl := &platform.VMTemplate{Name: "win", SourceType: "iso"}

	if !s.canDeployViaOperator(linuxTmpl, DeployVMInput{}, nil) {
		t.Fatal("expected container deploy via operator")
	}
	if s.canDeployViaOperator(isoTmpl, DeployVMInput{}, nil) {
		t.Fatal("iso should use hypervisor path")
	}
	if s.canDeployViaOperator(linuxTmpl, DeployVMInput{PublicIP: true}, nil) {
		t.Fatal("public IP should use hypervisor path")
	}
	if s.canDeployViaOperator(linuxTmpl, DeployVMInput{}, []string{"net-1"}) {
		t.Fatal("extra networks should use hypervisor path")
	}
}
