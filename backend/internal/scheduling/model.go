package scheduling

type Variable struct {
	Name   string
	Domain []any
}

type Assignment struct {
	Values  map[string]any
	Score   int
	Reasons []string
}

func (a Assignment) Value(name string) any {
	return a.Values[name]
}

type Constraint func(Assignment) (bool, error)

type Objective func(Assignment) (int, []string, error)

type Model struct {
	Variables   []Variable
	Constraints []Constraint
	Objective   Objective
	DedupKey    func(Assignment) string
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) AddVariable(name string, domain []any) {
	m.Variables = append(m.Variables, Variable{Name: name, Domain: domain})
}

func (m *Model) AddConstraint(c Constraint) {
	m.Constraints = append(m.Constraints, c)
}

func (m *Model) SetObjective(o Objective) {
	m.Objective = o
}

func (m *Model) SetDedupKey(fn func(Assignment) string) {
	m.DedupKey = fn
}
