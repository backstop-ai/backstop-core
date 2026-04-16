// Package distribution implements the pack distribution lifecycle: six
// commands (pack add, pack remove, pack install, pack update, pack upgrade,
// pack list) plus the backstop.lock format, SHA-256 content hashing,
// gate-time lock verification, tool_config provenance tracking, and
// .backstop/packs/ lifecycle management.
package distribution
