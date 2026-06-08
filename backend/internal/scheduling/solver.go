package scheduling

import "sort"

type Solver struct{}

func (e *CLPEngine) newSolver() *Solver {
	return &Solver{}
}

func (s *Solver) Solve(model Model, topN int) []Assignment {
	all := generateAll(model.Variables)

	var valid []Assignment
	for _, a := range all {
		pass := true
		for _, c := range model.Constraints {
			if !c(a) {
				pass = false
				break
			}
		}
		if pass {
			if model.Objective != nil {
				a.Score, a.Reasons = model.Objective(a)
			}
			valid = append(valid, a)
		}
	}

	sort.Slice(valid, func(i, j int) bool {
		if valid[i].Score != valid[j].Score {
			return valid[i].Score > valid[j].Score
		}
		return i < j
	})

	if model.DedupKey != nil {
		seen := make(map[string]bool)
		var deduped []Assignment
		for _, a := range valid {
			if key := model.DedupKey(a); !seen[key] {
				seen[key] = true
				deduped = append(deduped, a)
			}
		}
		valid = deduped
	}

	if len(valid) > topN {
		valid = valid[:topN]
	}
	return valid
}

func generateAll(vars []Variable) []Assignment {
	if len(vars) == 0 {
		return []Assignment{{Values: make(map[string]any)}}
	}

	head := vars[0]
	tail := generateAll(vars[1:])

	var result []Assignment
	for _, v := range head.Domain {
		for _, t := range tail {
			a := Assignment{Values: make(map[string]any)}
			for k, val := range t.Values {
				a.Values[k] = val
			}
			a.Values[head.Name] = v
			result = append(result, a)
		}
	}
	return result
}
