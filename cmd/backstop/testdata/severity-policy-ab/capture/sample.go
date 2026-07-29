package sample

// Trigger holds the single line the capture rule matches. Keep this file at
// exactly one match: the severity contract harness hard-fatals unless the
// dispatch yields exactly one violation.
func Trigger() {
	panic("boom")
}
