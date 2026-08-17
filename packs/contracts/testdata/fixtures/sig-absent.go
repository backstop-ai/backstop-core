package fix

// The CLEAN fixture for contract-signature. This pack's self-test pattern
// `func $NAME($$$PARAMS)` is name-agnostic, so ANY function declaration at all would
// fire it — the clean case must therefore declare none. A name-specific contrast file
// cannot serve as the clean case under a name-agnostic pattern.

var placeholder = "no functions declared here"
