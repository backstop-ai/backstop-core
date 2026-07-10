package app

// Risky reports the waivable finding (fake engine reports it at line 5).
func Risky() int {
	return risky() // WAIVABLE_DEFECT @waiver:backstop/waiver-e2e/waivable-defect:accepted-risk:2999-01-01
}

// Danger reports the protected finding (fake engine reports it at line 10).
func Danger() int {
	return danger() // PROTECTED_DEFECT @waiver:backstop/waiver-e2e/protected-defect:accepted-risk:2999-01-01
}

func risky() int  { return 1 }
func danger() int { return 2 }
