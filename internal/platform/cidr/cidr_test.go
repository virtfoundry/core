package cidr

import "testing"

func TestValidateVPCOverlap(t *testing.T) {
	if err := ValidateVPC("10.0.0.0/16", []string{"10.0.0.0/16"}); err == nil {
		t.Fatal("expected overlap error")
	}
	if err := ValidateVPC("10.1.0.0/16", []string{"10.0.0.0/16"}); err != nil {
		t.Fatalf("expected no overlap: %v", err)
	}
}

func TestValidateSubnetInsideVPC(t *testing.T) {
	if err := ValidateSubnet("10.0.0.0/16", "10.0.1.0/24", nil); err != nil {
		t.Fatalf("expected valid subnet: %v", err)
	}
	if err := ValidateSubnet("10.0.0.0/16", "10.1.0.0/24", nil); err == nil {
		t.Fatal("expected outside vpc error")
	}
}

func TestAllocateSubnet(t *testing.T) {
	auto, err := AllocateSubnet("10.0.0.0/16", []string{"10.0.0.0/24"}, 24)
	if err != nil {
		t.Fatal(err)
	}
	if auto == "10.0.0.0/24" {
		t.Fatalf("expected next free subnet, got %s", auto)
	}
}
