package sample

// Trips BOTH ast-grep rules in one scan so a single multi-rule invocation
// surfaces findings from rule-one AND rule-two.
func sample() {
	forbiddenCallOne(1)
	forbiddenCallTwo(2)
}
