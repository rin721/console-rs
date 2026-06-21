package initcenter

import (
	"fmt"
	"sort"
)

type InitTask interface {
	Definition() stepDefinition
}

type taskAdapter struct {
	def stepDefinition
}

func (t taskAdapter) Definition() stepDefinition {
	return t.def
}

type InitTaskRegistry struct {
	byKey map[string]stepDefinition
}

func NewInitTaskRegistry() *InitTaskRegistry {
	return &InitTaskRegistry{byKey: map[string]stepDefinition{}}
}

func (r *InitTaskRegistry) Register(task InitTask) error {
	if r == nil {
		return fmt.Errorf("init task registry is nil")
	}
	def := task.Definition()
	if def.Key == "" {
		return fmt.Errorf("init task key is required")
	}
	if _, ok := r.byKey[def.Key]; ok {
		return fmt.Errorf("duplicate init task %s", def.Key)
	}
	r.byKey[def.Key] = def
	return nil
}

func (r *InitTaskRegistry) Get(key string) (stepDefinition, bool) {
	if r == nil {
		return stepDefinition{}, false
	}
	def, ok := r.byKey[key]
	return def, ok
}

func (r *InitTaskRegistry) Resolve() ([]stepDefinition, error) {
	if r == nil {
		return nil, fmt.Errorf("init task registry is nil")
	}
	resolver := InitDependencyResolver{tasks: r.byKey}
	return resolver.Resolve()
}

type InitDependencyResolver struct {
	tasks map[string]stepDefinition
}

func (r InitDependencyResolver) Resolve() ([]stepDefinition, error) {
	keys := make([]string, 0, len(r.tasks))
	for key := range r.tasks {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := r.tasks[keys[i]]
		right := r.tasks[keys[j]]
		if left.Order == right.Order {
			return left.Key < right.Key
		}
		return left.Order < right.Order
	})

	visited := map[string]int{}
	out := make([]stepDefinition, 0, len(keys))
	var visit func(string) error
	visit = func(key string) error {
		switch visited[key] {
		case 1:
			return fmt.Errorf("cyclic init task dependency at %s", key)
		case 2:
			return nil
		}
		def, ok := r.tasks[key]
		if !ok {
			return fmt.Errorf("missing init task dependency %s", key)
		}
		visited[key] = 1
		deps := append([]string(nil), def.Dependencies...)
		sort.Slice(deps, func(i, j int) bool {
			left := r.tasks[deps[i]]
			right := r.tasks[deps[j]]
			if left.Order == right.Order {
				return deps[i] < deps[j]
			}
			return left.Order < right.Order
		})
		for _, dep := range deps {
			if _, ok := r.tasks[dep]; !ok {
				return fmt.Errorf("task %s depends on missing task %s", key, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		visited[key] = 2
		out = append(out, def)
		return nil
	}
	for _, key := range keys {
		if err := visit(key); err != nil {
			return nil, err
		}
	}
	return out, nil
}
