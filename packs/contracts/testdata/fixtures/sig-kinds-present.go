package fix

type CheckType int

type Result struct {
	Name string
	OK   bool
}

const CheckTypeFindings = "findings"

var GlobalRegistry = map[string]int{}

func (ct CheckType) String() string { return "ct" }

type Stringer interface {
	String() string
}
