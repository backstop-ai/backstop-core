package fix

func String() string { return "free" }

type Other struct{}

func (x Other) String() string { return "other" }

type SomethingElse int
