package engine

import "sort"

// Registry holds every algorithm the backend knows how to run, keyed by ID.
type Registry struct {
	algorithms map[string]Algorithm
}

func NewRegistry() *Registry {
	return &Registry{algorithms: make(map[string]Algorithm)}
}

func (r *Registry) Register(a Algorithm) {
	r.algorithms[a.ID()] = a
}

func (r *Registry) Get(id string) (Algorithm, bool) {
	a, ok := r.algorithms[id]
	return a, ok
}

// Summary is the lightweight listing shown before a user picks an algorithm.
type Summary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Background  string `json:"background"`
}

// List returns every registered algorithm's summary, sorted by ID for a
// stable ordering in the UI.
func (r *Registry) List() []Summary {
	out := make([]Summary, 0, len(r.algorithms))
	for _, a := range r.algorithms {
		out = append(out, Summary{ID: a.ID(), Title: a.Title(), Description: a.Description(), Background: a.Background()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
