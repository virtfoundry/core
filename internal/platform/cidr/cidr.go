package cidr

import (
	"fmt"
	"math/big"
	"net"
	"sort"
)

// RFC1918 private ranges (GCP-compatible primary ranges).
var rfc1918Supernets = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

// Common VPC sizes offered in the UI (GCP custom mode uses /16 or larger).
var vpcPrefixSuggestions = []int{16, 20, 22, 24}

// Subnet prefix lengths (GCP supports down to /29; we stop at /28 for practical limits).
var subnetPrefixSuggestions = []int{24, 25, 26, 27, 28}

// Block describes a CIDR allocation option.
type Block struct {
	CIDR      string `json:"cidr"`
	Label     string `json:"label"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// VPCPlan is returned when creating a VPC.
type VPCPlan struct {
	Suggestions []Block  `json:"suggestions"`
	Existing    []string `json:"existing"`
}

// SubnetPlan is returned when creating a subnet inside a VPC.
type SubnetPlan struct {
	VPCCIDR     string   `json:"vpc_cidr"`
	Prefix      int      `json:"prefix"`
	Auto        string   `json:"auto,omitempty"`
	Suggestions []Block  `json:"suggestions"`
	Used        []string `json:"used"`
}

func Parse(s string) (*net.IPNet, error) {
	ip, n, err := net.ParseCIDR(s)
	if err != nil {
		return nil, fmt.Errorf("invalid cidr %q: %w", s, err)
	}
	n.IP = ip.Mask(n.Mask)
	return n, nil
}

func IsPrivateRFC1918(cidr string) bool {
	n, err := Parse(cidr)
	if err != nil {
		return false
	}
	for _, sup := range rfc1918Supernets {
		parent, _ := Parse(sup)
		if Contains(parent, n) {
			return true
		}
	}
	return false
}

func Contains(parent, child *net.IPNet) bool {
	pOnes, _ := parent.Mask.Size()
	cOnes, _ := child.Mask.Size()
	if cOnes < pOnes {
		return false
	}
	return parent.Contains(child.IP)
}

func Overlap(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP) || a.Contains(lastIP(b)) || b.Contains(lastIP(a))
}

func lastIP(n *net.IPNet) net.IP {
	ip := n.IP.To4()
	if ip == nil {
		return nil
	}
	mask := n.Mask
	out := make(net.IP, len(ip))
	for i := range ip {
		out[i] = ip[i] | ^mask[i]
	}
	return out
}

func ValidateVPC(cidr string, existing []string) error {
	n, err := Parse(cidr)
	if err != nil {
		return err
	}
	if !IsPrivateRFC1918(cidr) {
		return fmt.Errorf("vpc cidr must be a private RFC1918 range (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)")
	}
	ones, bits := n.Mask.Size()
	if bits-ones < 8 {
		return fmt.Errorf("vpc cidr too small — use /16 or larger network")
	}
	for _, ex := range existing {
		other, err := Parse(ex)
		if err != nil {
			continue
		}
		if Overlap(n, other) {
			return fmt.Errorf("vpc cidr overlaps existing vpc %s", ex)
		}
	}
	return nil
}

func ValidateSubnet(vpcCIDR, subnetCIDR string, existing []string) error {
	vpc, err := Parse(vpcCIDR)
	if err != nil {
		return fmt.Errorf("invalid vpc cidr")
	}
	sub, err := Parse(subnetCIDR)
	if err != nil {
		return err
	}
	if !Contains(vpc, sub) {
		return fmt.Errorf("subnet must be inside vpc range %s", vpcCIDR)
	}
	vpcOnes, _ := vpc.Mask.Size()
	subOnes, _ := sub.Mask.Size()
	if subOnes < vpcOnes {
		return fmt.Errorf("subnet cannot be broader than vpc")
	}
	for _, ex := range existing {
		other, err := Parse(ex)
		if err != nil {
			continue
		}
		if Overlap(sub, other) {
			return fmt.Errorf("subnet overlaps existing network %s", ex)
		}
	}
	return nil
}

// PlanVPC suggests non-overlapping /16 blocks (GCP-style pick-your-range).
func PlanVPC(existing []string) VPCPlan {
	existingNets := parseAll(existing)
	var suggestions []Block

	// 10.0.0.0/8 — offer /16 slots (most common GCP pattern).
	for i := 0; i < 256; i++ {
		candidate := fmt.Sprintf("10.%d.0.0/16", i)
		suggestions = append(suggestions, evalBlock(candidate, "Rede privada 10.x", existingNets))
		if len(suggestions) >= 8 {
			break
		}
	}

	// 172.16.0.0/12 — /16 slots.
	for i := 16; i < 32; i++ {
		candidate := fmt.Sprintf("172.%d.0.0/16", i)
		suggestions = append(suggestions, evalBlock(candidate, "Rede privada 172.x", existingNets))
	}

	// 192.168.0.0/16 subdivisions /24 as VPC (small envs).
	for i := 0; i < 256; i += 16 {
		candidate := fmt.Sprintf("192.168.%d.0/20", i)
		suggestions = append(suggestions, evalBlock(candidate, "Rede privada 192.168.x", existingNets))
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Available != suggestions[j].Available {
			return suggestions[i].Available
		}
		return suggestions[i].CIDR < suggestions[j].CIDR
	})

	// Return first 12 with at least 6 available shown first.
	out := make([]Block, 0, 12)
	for _, s := range suggestions {
		if s.Available {
			out = append(out, s)
		}
		if len(out) >= 8 {
			break
		}
	}
	for _, s := range suggestions {
		if !s.Available && len(out) < 12 {
			out = append(out, s)
		}
	}

	return VPCPlan{Suggestions: out, Existing: existing}
}

// PlanSubnet suggests subnets inside a VPC (auto + manual options).
func PlanSubnet(vpcCIDR string, existing []string, prefix int) (SubnetPlan, error) {
	if prefix <= 0 {
		prefix = 24
	}
	vpc, err := Parse(vpcCIDR)
	if err != nil {
		return SubnetPlan{}, err
	}
	existingNets := parseAll(existing)

	plan := SubnetPlan{
		VPCCIDR: vpcCIDR,
		Prefix:  prefix,
		Used:    existing,
	}

	auto, err := AllocateSubnet(vpcCIDR, existing, prefix)
	if err == nil {
		plan.Auto = auto
	}

	// Walk VPC space in prefix-sized steps (FIRST_SMALLEST_FITTING style).
	hostBits := 32 - prefix
	step := big.NewInt(1)
	step.Lsh(step, uint(hostBits))

	vpcStart := ipToInt(vpc.IP.To4())
	vpcEnd := ipToInt(lastIP(vpc).To4())

	var suggestions []Block
	for cur := new(big.Int).Set(vpcStart); cur.Cmp(vpcEnd) <= 0; cur.Add(cur, step) {
		ip := intToIP(cur)
		mask := net.CIDRMask(prefix, 32)
		n := &net.IPNet{IP: ip.Mask(mask), Mask: mask}
		candidate := n.String()
		block := evalBlock(candidate, fmt.Sprintf("/%d dentro da VPC", prefix), existingNets)
		if !Contains(vpc, n) {
			block.Available = false
			block.Reason = "fora da VPC"
		}
		suggestions = append(suggestions, block)
		if len(suggestions) >= 16 {
			break
		}
	}

	plan.Suggestions = suggestions
	return plan, nil
}

// AllocateSubnet finds the first free block of prefix length inside VPC (GCP auto mode).
func AllocateSubnet(vpcCIDR string, existing []string, prefix int) (string, error) {
	vpc, err := Parse(vpcCIDR)
	if err != nil {
		return "", err
	}
	existingNets := parseAll(existing)
	hostBits := 32 - prefix
	step := big.NewInt(1)
	step.Lsh(step, uint(hostBits))

	vpcStart := ipToInt(vpc.IP.To4())
	vpcEnd := ipToInt(lastIP(vpc).To4())

	for cur := new(big.Int).Set(vpcStart); cur.Cmp(vpcEnd) <= 0; cur.Add(cur, step) {
		ip := intToIP(cur)
		mask := net.CIDRMask(prefix, 32)
		n := &net.IPNet{IP: ip.Mask(mask), Mask: mask}
		if !Contains(vpc, n) {
			continue
		}
		if overlapsAny(n, existingNets) {
			continue
		}
		return n.String(), nil
	}
	return "", fmt.Errorf("no free /%d subnet in vpc %s", prefix, vpcCIDR)
}

func VPCPrefixOptions() []int  { return append([]int{}, vpcPrefixSuggestions...) }
func SubnetPrefixOptions() []int { return append([]int{}, subnetPrefixSuggestions...) }

func parseAll(cidrs []string) []*net.IPNet {
	var out []*net.IPNet
	for _, c := range cidrs {
		if n, err := Parse(c); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func overlapsAny(n *net.IPNet, existing []*net.IPNet) bool {
	for _, ex := range existing {
		if Overlap(n, ex) {
			return true
		}
	}
	return false
}

func evalBlock(candidate, label string, existing []*net.IPNet) Block {
	n, err := Parse(candidate)
	if err != nil {
		return Block{CIDR: candidate, Label: label, Available: false, Reason: "inválido"}
	}
	if !IsPrivateRFC1918(candidate) {
		return Block{CIDR: candidate, Label: label, Available: false, Reason: "não RFC1918"}
	}
	for _, ex := range existing {
		if Overlap(n, ex) {
			return Block{CIDR: candidate, Label: label, Available: false, Reason: "em uso"}
		}
	}
	return Block{CIDR: candidate, Label: label, Available: true}
}

func ipToInt(ip net.IP) *big.Int {
	v := big.NewInt(0)
	if ip4 := ip.To4(); ip4 != nil {
		v.SetBytes(ip4)
	}
	return v
}

func intToIP(v *big.Int) net.IP {
	b := v.Bytes()
	padded := make([]byte, 4)
	copy(padded[4-len(b):], b)
	return net.IP(padded)
}
