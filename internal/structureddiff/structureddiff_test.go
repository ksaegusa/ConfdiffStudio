package structureddiff

import (
	"strings"
	"testing"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func lines(text string) []string {
	return strings.Split(text, "\n")
}

func TestBuildFileLenientIgnoresTopLevelReordering(t *testing.T) {
	before := `hostname edge-01
ip route 10.0.0.0 255.255.255.0 192.0.2.1
ip route 10.1.0.0 255.255.255.0 192.0.2.2`
	after := `hostname edge-01
ip route 10.1.0.0 255.255.255.0 192.0.2.2
ip route 10.0.0.0 255.255.255.0 192.0.2.1`

	file, err := BuildFile("fw.cfg", lines(before), lines(after), model.DiffProfile{OrderMode: "lenient"})
	if err != nil {
		t.Fatalf("BuildFile returned error: %v", err)
	}
	if file.Changed {
		t.Fatalf("expected no diff for top-level reorder, got %+v", file.Blocks)
	}
}

func TestBuildFileStrictDetectsSiblingReordering(t *testing.T) {
	before := `ip access-list extended LAN-OUT
 permit udp any host 8.8.8.8 eq 53
 permit tcp 10.0.0.0 0.0.0.255 any eq 80`
	after := `ip access-list extended LAN-OUT
 permit tcp 10.0.0.0 0.0.0.255 any eq 80
 permit udp any host 8.8.8.8 eq 53`

	file, err := BuildFile("fw.cfg", lines(before), lines(after), model.DiffProfile{OrderMode: "strict"})
	if err != nil {
		t.Fatalf("BuildFile returned error: %v", err)
	}
	if !file.Changed {
		t.Fatal("expected strict mode to detect reorder")
	}
	if len(file.Blocks) == 0 {
		t.Fatal("expected blocks to be present")
	}
}

func TestBuildFileLenientDetectsNestedReordering(t *testing.T) {
	before := `ip access-list extended LAN-OUT
 permit udp any host 8.8.8.8 eq 53
 permit tcp 10.0.0.0 0.0.0.255 any eq 80`
	after := `ip access-list extended LAN-OUT
 permit tcp 10.0.0.0 0.0.0.255 any eq 80
 permit udp any host 8.8.8.8 eq 53`

	file, err := BuildFile("fw.cfg", lines(before), lines(after), model.DiffProfile{OrderMode: "lenient"})
	if err != nil {
		t.Fatalf("BuildFile returned error: %v", err)
	}
	if !file.Changed {
		t.Fatal("expected lenient mode to detect reorder inside a matched block")
	}
	if len(file.Blocks) == 0 {
		t.Fatal("expected blocks to be present")
	}
	if file.Blocks[0].ContextPath[0] != "ip access-list extended LAN-OUT" {
		t.Fatalf("expected ACL context path, got %+v", file.Blocks[0].ContextPath)
	}
}

func TestBuildFileLenientBuildsAncestorContext(t *testing.T) {
	before := `interface GigabitEthernet0/0
 description uplink
 ip address 192.0.2.1 255.255.255.252`
	after := `interface GigabitEthernet0/0
 description uplink
 ip address 192.0.2.2 255.255.255.252`

	file, err := BuildFile("fw.cfg", lines(before), lines(after), model.DiffProfile{OrderMode: "lenient"})
	if err != nil {
		t.Fatalf("BuildFile returned error: %v", err)
	}
	if len(file.Blocks) != 1 {
		t.Fatalf("expected a merged block, got %+v", file.Blocks)
	}
	block := file.Blocks[0]
	if len(block.ContextPath) != 1 || block.ContextPath[0] != "interface GigabitEthernet0/0" {
		t.Fatalf("unexpected context path: %+v", block.ContextPath)
	}
	if len(block.BeforeLines) != 1 || len(block.AfterLines) != 1 {
		t.Fatalf("expected one before and one after line, got %+v", block)
	}
	if block.Kind != model.StructuredDiffBlockChange {
		t.Fatalf("expected change block, got %s", block.Kind)
	}
}

func TestBuildFileTargetPrefixesLimitScope(t *testing.T) {
	before := `interface GigabitEthernet0/0
 description uplink
ip route 0.0.0.0 0.0.0.0 192.0.2.2`
	after := `interface GigabitEthernet0/0
 description uplink
ip route 0.0.0.0 0.0.0.0 192.0.2.254`

	file, err := BuildFile("fw.cfg", lines(before), lines(after), model.DiffProfile{
		OrderMode:      "lenient",
		TargetPrefixes: []string{"interface"},
	})
	if err != nil {
		t.Fatalf("BuildFile returned error: %v", err)
	}
	if file.Changed {
		t.Fatalf("expected route change to be ignored, got %+v", file.Blocks)
	}
}

func TestCompileProfileRejectsDangerousIgnoreRegex(t *testing.T) {
	_, err := CompileProfile(model.DiffProfile{IgnorePatterns: []string{"^"}})
	if err == nil {
		t.Fatal("expected error for empty-string-matching regex")
	}
}

func TestBuildFileClassifiesAddAndRemoveBlocks(t *testing.T) {
	addFile, err := BuildFile("fw.cfg", lines("hostname edge-01"), lines("hostname edge-01\ninterface Gi0/0"), model.DiffProfile{})
	if err != nil {
		t.Fatalf("BuildFile add returned error: %v", err)
	}
	if len(addFile.Blocks) != 1 || addFile.Blocks[0].Kind != model.StructuredDiffBlockAdd {
		t.Fatalf("expected add block, got %+v", addFile.Blocks)
	}

	removeFile, err := BuildFile("fw.cfg", lines("hostname edge-01\ninterface Gi0/0"), lines("hostname edge-01"), model.DiffProfile{})
	if err != nil {
		t.Fatalf("BuildFile remove returned error: %v", err)
	}
	if len(removeFile.Blocks) != 1 || removeFile.Blocks[0].Kind != model.StructuredDiffBlockRemove {
		t.Fatalf("expected remove block, got %+v", removeFile.Blocks)
	}
}
