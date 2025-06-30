package datasource

type UnionFind struct {
	parent map[string]string
}

func NewUnionFind() *UnionFind {
	return &UnionFind{parent: make(map[string]string)}
}

func (uf *UnionFind) Find(x string) string {
	if uf.parent[x] == "" {
		uf.parent[x] = x
	}
	if uf.parent[x] != x {
		uf.parent[x] = uf.Find(uf.parent[x])
	}
	return uf.parent[x]
}

func (uf *UnionFind) Union(x, y string) {
	rootX := uf.Find(x)
	rootY := uf.Find(y)
	if rootX != rootY {
		uf.parent[rootY] = rootX
	}
}
