package trees

import (
	"strings"
	"testing"

	"github.com/heldtogether/traintrack/internal/datasets"
)

func TestRenderTree2Commits(t *testing.T) {
	c := []*datasets.Dataset{
		{ID: "A", Parent: stringPtr("")},
		{ID: "B", Parent: stringPtr("A")},
	}

	expected := `* A
* B`

	tree := BuildTree(c)
	out := strings.Join(RenderTree(tree, "", ""), "\n")
	if out != expected {
		t.Errorf("fail: wanted\n%s\ngot\n%s\n", expected, out)
	}
}

func TestRenderTree3Commits(t *testing.T) {
	c := []*datasets.Dataset{
		{ID: "A", Parent: stringPtr("")},
		{ID: "B", Parent: stringPtr("A")},
		{ID: "C", Parent: stringPtr("B")},
	}

	expected :=
		`* A
* B
* C`

	tree := BuildTree(c)
	out := strings.Join(RenderTree(tree, "", ""), "\n")
	if out != expected {
		t.Errorf("fail: wanted\n%s\ngot\n%s\n", expected, out)
	}
}

func TestRenderTree1Branch(t *testing.T) {
	c := []*datasets.Dataset{
		{ID: "A", Parent: stringPtr("")},
		{ID: "B", Parent: stringPtr("A")},
		{ID: "C", Parent: stringPtr("A")},
		{ID: "D", Parent: stringPtr("B")},
	}

	expected :=
		`* A
|\
| * B
| * D
* C`

	tree := BuildTree(c)
	out := strings.Join(RenderTree(tree, "", ""), "\n")
	if out != expected {
		t.Errorf("fail: wanted\n%s\ngot\n%s\n", expected, out)
	}
}

func TestRenderTree2Branch(t *testing.T) {
	c := []*datasets.Dataset{
		{ID: "A", Parent: nil},
		{ID: "B", Parent: stringPtr("A")},
		{ID: "C", Parent: stringPtr("A")},
		{ID: "D", Parent: stringPtr("B")},
		{ID: "E", Parent: stringPtr("B")},
		{ID: "F", Parent: stringPtr("D")},
	}

	expected :=
		`* A
|\
| * B
| |\
| | * D
| | * F
| * E
* C`

	tree := BuildTree(c)
	out := strings.Join(RenderTree(tree, "", ""), "\n")
	if out != expected {
		t.Errorf("fail: wanted\n%s\ngot\n%s\n", expected, out)
	}
}

func TestRenderTree2Branch1Parent(t *testing.T) {
	c := []*datasets.Dataset{
		{ID: "A", Parent: nil},
		{ID: "B", Parent: stringPtr("A")},
		{ID: "C", Parent: stringPtr("A")},
		{ID: "D", Parent: stringPtr("A")},
	}

	expected :=
		`* A
|\ \
| | * B
| * C
* D`

	tree := BuildTree(c)
	out := strings.Join(RenderTree(tree, "", ""), "\n")
	if out != expected {
		t.Errorf("fail: wanted\n%s\ngot\n%s\n", expected, out)
	}
}

func TestRenderTree2Roots(t *testing.T) {
	c := []*datasets.Dataset{
		{ID: "A", Parent: nil},
		{ID: "B", Parent: nil},
	}

	expected :=
		`* A

---

* B`

	tree := BuildTree(c)
	out := strings.Join(RenderTree(tree, "", ""), "\n")
	if out != expected {
		t.Errorf("fail: wanted\n%s\ngot\n%s\n", expected, out)
	}
}
