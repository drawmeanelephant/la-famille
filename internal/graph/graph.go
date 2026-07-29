package graph

type Node struct {
	Type         string   `json:"type"`
	ReferencedBy []string `json:"referenced_by,omitempty"`
	Render       bool     `json:"render"`
	Missing      bool     `json:"missing,omitempty"`
}

type Graph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges [][2]string     `json:"edges"`
}
