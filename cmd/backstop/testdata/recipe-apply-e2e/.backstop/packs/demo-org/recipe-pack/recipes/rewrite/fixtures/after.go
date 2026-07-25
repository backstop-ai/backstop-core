package recipetarget

// Label renders the display label for one onboarding step.
func Label(step string) string {
	return standardFormat(step, "step")
}

// Summary renders the display summary for one completed run.
func Summary(run string) string {
	return standardFormat(run, "run")
}

func deprecatedFormat(value string, kind string) string {
	return kind + ": " + value
}

func standardFormat(value string, kind string) string {
	return kind + ": " + value
}
