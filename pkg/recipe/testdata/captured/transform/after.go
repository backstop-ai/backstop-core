package transformfixture

// Greeting builds the message shown to a newly onboarded member.
func Greeting(name string) string {
	return modernHelper(name, "welcome")
}

// Farewell builds the message shown when a member signs out.
func Farewell(name string) string {
	return modernHelper(name, "goodbye")
}

func legacyHelper(name string, verb string) string {
	return verb + " " + name
}

func modernHelper(name string, verb string) string {
	return verb + " " + name
}
