package runtime

import (
	"strings"
	"testing"
)

func TestDependencyGraph_Empty(t *testing.T) {
	g := NewDependencyGraph()
	sorted, err := g.Sort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 0 {
		t.Errorf("expected empty slice, got %v", sorted)
	}
}

func TestDependencyGraph_SingleNode(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")
	sorted, err := g.Sort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 1 || sorted[0] != "A" {
		t.Errorf("expected [A], got %v", sorted)
	}
}

func TestDependencyGraph_LinearChain(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge("A", "B") // A depends on B
	g.AddEdge("B", "C") // B depends on C

	sorted, err := g.Sort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(sorted))
	}

	// C must come before B, B must come before A
	indexOf := make(map[string]int)
	for i, n := range sorted {
		indexOf[n] = i
	}
	if indexOf["C"] >= indexOf["B"] {
		t.Errorf("C should come before B: %v", sorted)
	}
	if indexOf["B"] >= indexOf["A"] {
		t.Errorf("B should come before A: %v", sorted)
	}
}

func TestDependencyGraph_Diamond(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge("A", "B") // A depends on B
	g.AddEdge("A", "C") // A depends on C
	g.AddEdge("B", "D") // B depends on D
	g.AddEdge("C", "D") // C depends on D

	sorted, err := g.Sort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(sorted))
	}

	indexOf := make(map[string]int)
	for i, n := range sorted {
		indexOf[n] = i
	}
	// D must come before B and C, both must come before A
	if indexOf["D"] >= indexOf["B"] {
		t.Errorf("D should come before B: %v", sorted)
	}
	if indexOf["D"] >= indexOf["C"] {
		t.Errorf("D should come before C: %v", sorted)
	}
	if indexOf["B"] >= indexOf["A"] {
		t.Errorf("B should come before A: %v", sorted)
	}
	if indexOf["C"] >= indexOf["A"] {
		t.Errorf("C should come before A: %v", sorted)
	}
}

func TestDependencyGraph_IndependentNodes(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")

	sorted, err := g.Sort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(sorted))
	}

	// All three should be present (order doesn't matter)
	found := make(map[string]bool)
	for _, n := range sorted {
		found[n] = true
	}
	for _, name := range []string{"A", "B", "C"} {
		if !found[name] {
			t.Errorf("missing node %s in sorted output", name)
		}
	}
}

func TestDependencyGraph_CycleDetection_TwoNode(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge("A", "B")
	g.AddEdge("B", "A")

	_, err := g.Sort()
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected circular dependency error, got %q", err.Error())
	}
}

func TestDependencyGraph_CycleDetection_ThreeNode(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("C", "A")

	_, err := g.Sort()
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected circular dependency error, got %q", err.Error())
	}
}

func TestDependencyGraph_AddNode_HasNode(t *testing.T) {
	g := NewDependencyGraph()

	if g.HasNode("A") {
		t.Error("expected HasNode to return false for non-existent node")
	}

	g.AddNode("A")
	if !g.HasNode("A") {
		t.Error("expected HasNode to return true after AddNode")
	}
}

func TestDependencyGraph_AddEdge_ImplicitNodes(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge("A", "B")

	if !g.HasNode("A") {
		t.Error("expected A to be added implicitly by AddEdge")
	}
	if !g.HasNode("B") {
		t.Error("expected B to be added implicitly by AddEdge")
	}
}

func TestDependencyGraph_GetDependencies(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")

	deps := g.GetDependencies("A")
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	found := make(map[string]bool)
	for _, d := range deps {
		found[d] = true
	}
	if !found["B"] || !found["C"] {
		t.Errorf("expected dependencies [B, C], got %v", deps)
	}
}

func TestDependencyGraph_GetDependencies_NoDeps(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("A")

	deps := g.GetDependencies("A")
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(deps))
	}
}
