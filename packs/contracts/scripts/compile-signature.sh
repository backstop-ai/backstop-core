#!/bin/sh
# contract->ast-grep-pattern COMPILER (SPEC-038 REQ-002/CLM-006; ISSUE-036). This
# is a PACK-RELATIVE script — the backstop binary NEVER compiles or renders a
# signature; it passes the human-readable Signature here and reads back an
# ast-grep pattern. The compiler is DECLARATION-KIND-AWARE: it infers the Go
# declaration kind from the LEADING TOKEN of the signature text (the runtime
# passes ONLY the signature string, never the contract's `kind` field) and emits
# the matching STRUCTURAL ast-grep pattern per kind. Every pattern is
# param-NAME/whitespace-insensitive (CLM-007) while preserving the declared
# identity (name, receiver type, return clause). ast-grep knows nothing of Go
# signatures; backstop knows nothing of ast-grep patterns — this script is the
# only place the mapping lives.
#
#   func   func RouteFile(path string, mode int) (string, error)
#          -> func RouteFile($$$PARAMS) (string, error)
#   method func (ct CheckType) String() string
#          -> func ($R CheckType) String($$$PARAMS) string
#          (receiver TYPE preserved incl. a leading '*' for pointer receivers;
#           only the receiver NAME + params are metavar'd — so it will NOT match a
#           same-named free function nor a method on a DIFFERENT receiver type.)
#   type   type CheckType int / type Result struct {…}
#          -> type CheckType $$$
#   iface  type Stringer interface { String() string }
#          -> type Stringer interface { $$$ }
#   const  const CheckTypeFindings = "findings"
#          -> const CheckTypeFindings = $$$   (the '=' is REQUIRED — a bare
#           `const $NAME $$$` is an ast-grep ERROR node that matches nothing.)
#   var    var GlobalRegistry = map[string]int{}  -> var GlobalRegistry = $$$
#          var Foo int                            -> var Foo int
#
# The signature is read from $1 (or stdin if $1 is empty).
sig="$1"
if [ -z "$sig" ]; then
  sig=$(cat)
fi

# trim leading+trailing whitespace of $1
trim() {
  printf '%s' "$1" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//'
}

# compile a func-shaped body ("<name>(<params>)<rest>") into
# "func <name>($$$PARAMS) <rest>" (rest = return clause, possibly empty).
emit_func() {
  body=$(trim "$1")
  fname=$(trim "${body%%(*}")
  fret=$(trim "${body#*\)}")
  if [ -n "$fret" ]; then
    printf 'func %s($$$PARAMS) %s' "$fname" "$fret"
  else
    printf 'func %s($$$PARAMS)' "$fname"
  fi
}

case "$sig" in
  "func ("*)
    # METHOD: func (<recv> <recvType>) <name>(<params>) <ret>
    afterfunc=$(trim "${sig#func }")            # "(ct CheckType) String() string"
    recv=${afterfunc#*\(}                        # "ct CheckType) String() string"
    recv=$(trim "${recv%%\)*}")                  # "ct CheckType"  (receiver clause)
    rest=$(trim "${afterfunc#*\)}")              # "String() string" (method body)
    # receiver = "<name> <type>"; preserve the TYPE (may carry a leading '*'),
    # metavar only the receiver NAME. A receiver with no name (rare) collapses to
    # just the type, which we still preserve verbatim.
    recvType=$(trim "${recv#* }")
    mname=$(trim "${rest%%(*}")
    mret=$(trim "${rest#*\)}")
    if [ -n "$mret" ]; then
      printf 'func ($R %s) %s($$$PARAMS) %s' "$recvType" "$mname" "$mret"
    else
      printf 'func ($R %s) %s($$$PARAMS)' "$recvType" "$mname"
    fi
    ;;
  "func "*)
    # FUNCTION: unchanged behavior.
    emit_func "${sig#func }"
    ;;
  "type "*)
    # TYPE or INTERFACE: name is the token after "type ".
    body=$(trim "${sig#type }")                  # "CheckType int" / "Stringer interface {…}"
    tname=${body%% *}
    case "$sig" in
      *interface*)
        printf 'type %s interface { $$$ }' "$tname"
        ;;
      *)
        printf 'type %s $$$' "$tname"
        ;;
    esac
    ;;
  "const "*)
    # CONSTANT: the '=' is REQUIRED (bare const $NAME $$$ is an ast-grep ERROR).
    body=$(trim "${sig#const }")                 # "CheckTypeFindings = \"findings\""
    cname=${body%% *}
    printf 'const %s = $$$' "$cname"
    ;;
  "var "*)
    # VARIABLE: preserve the RHS ("= $$$") when assigned, else the declared type.
    body=$(trim "${sig#var }")                   # "GlobalRegistry = …" / "Foo int"
    vname=${body%% *}
    case "$body" in
      *=*)
        printf 'var %s = $$$' "$vname"
        ;;
      *)
        vtype=$(trim "${body#* }")
        printf 'var %s %s' "$vname" "$vtype"
        ;;
    esac
    ;;
  *)
    # No leading keyword. Two shapes reach here:
    #   - STRUCT FIELD ("<Name> <Type> [`tag`]", no parens) — ISSUE-036: a bare field
    #     is an ast-grep ERROR node (not valid Go at top level), so wrap it in struct
    #     context to match the field within any struct in the target file.
    #   - bare func-shaped body ("<name>(<params>)<rest>", has parens) — back-compat.
    case "$sig" in
      *"("*)
        emit_func "$sig"
        ;;
      *)
        # NEWLINES + UNNAMED $$$ are load-bearing: ast-grep parses a struct field
        # list by line/semicolon, so `$$$ <field> $$$` on one line does NOT match, and
        # NAMED metavars over-match every field position. `$$$\n<field>\n$$$` matches
        # the containing struct exactly once.
        printf 'struct {\n$$$\n%s\n$$$\n}' "$(trim "$sig")"
        ;;
    esac
    ;;
esac
