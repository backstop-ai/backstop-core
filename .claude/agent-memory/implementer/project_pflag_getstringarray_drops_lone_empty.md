---
name: pflag-getstringarray-drops-lone-empty
description: pflag's GetStringArray CSV-round-trips the value and silently DROPS a lone empty entry, so `--flag ""` reads back as an EMPTY list; read Lookup(name).Value.(pflag.SliceValue).GetSlice() instead
metadata:
  type: project
---

`cmd.Flags().GetStringArray("x")` does NOT return the accumulated slice. It renders the
flag to its string form and re-parses it through `stringArrayConv`, which treats an
empty rendering as `[]string{}`. So `--x ""` comes back as an **empty list**, not `[""]`.

**Why it matters:** a guard written as "reject empty entries" iterating the
`GetStringArray` result is UNREACHABLE for the single-empty case — the very case it
exists to catch. In ISSUE-093 that meant `gate --file ""` still fell through to a
diff-scoped whole-repo sweep (DEFECT-3) while the refusal loop looked correct and
two of the three sub-cases (`--file a --file ""`, `--file "   "`) passed, which makes
the hole look like a passing test suite with one odd failure.

**How to apply:** when a repeated string flag's EMPTY values are semantically
meaningful (refusal, validation, count), read the exact slice:

```go
if sliceValue, ok := cmd.Flags().Lookup("file").Value.(pflag.SliceValue); ok {
    fileValues = sliceValue.GetSlice()
}
```

`stringArrayValue.GetSlice()` returns `*s.value` verbatim. Keep the `GetStringArray`
call for its type-mismatch error surface, then overwrite with `GetSlice()`.

Also: use `StringArrayVar`, never `StringSliceVar` — the latter splits on commas and
silently shreds a path containing one. See [[project_gate_file_flag_takes_positional_args]].
