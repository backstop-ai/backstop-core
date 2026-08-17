package fix

// The shared CLEAN fixture for all five per-kind signature rules. Their self-test
// patterns are name-agnostic, so this file declares NO type, NO `const NAME = ...`,
// NO `var NAME = ...`, NO method (no receiver) and NO interface — any instance at all
// would fire the matching pattern whatever name it carried.

func plainFunction() string { return "no kinds declared here" }
