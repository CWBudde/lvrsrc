// Command gen-heap-tags reads pylabview's LVheap.py and emits Go
// constants for every enum declared there. Output goes to
// internal/codecs/heap/tags_gen.go.
//
// Usage:
//
//	go run ./scripts/gen-heap-tags > internal/codecs/heap/tags_gen.go
//
// The generator is deliberately strict: any unparseable line in the
// source will surface as an error so we notice when pylabview's enum
// shape changes upstream.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// classRE matches `class Foo(BaseClass):` lines.
var classRE = regexp.MustCompile(`^class\s+([A-Za-z0-9_]+)\(([A-Za-z0-9_.]+)\):\s*$`)

// memberRE matches `Name = number` enum members. Comments after the
// number are tolerated.
var memberRE = regexp.MustCompile(`^\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(-?\d+)\s*(#.*)?$`)

// enumBaseClasses are the enum-style base classes whose members we
// extract. Anything else (TDObject, etc.) is ignored.
var enumBaseClasses = map[string]bool{
	"ENUM_TAGS":    true,
	"enum.Enum":    true,
	"enum.IntEnum": true,
}

// enumSpec is one parsed enum with its members.
type enumSpec struct {
	pyName  string         // e.g. SL_SYSTEM_TAGS
	goType  string         // e.g. SystemTag
	members []enumMember   // declaration order
	seen    map[string]int // member name -> value, for duplicate detection
}

type enumMember struct {
	pyName string
	goName string
	value  int
}

// goTypeFor maps a pylabview enum class name to a Go type name. The
// rules: drop trailing `_TAGS`, strip leading `SL_` or `OBJ_`, then
// CamelCase across underscores.
func goTypeFor(pyName string) string {
	s := strings.TrimSuffix(pyName, "_TAGS")
	s = strings.TrimPrefix(s, "SL_")
	s = strings.TrimPrefix(s, "OBJ_")
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		// Lowercase the rest after the first letter so we get
		// `MultiDim` rather than `MULTI_DIM` or `MULTIDIM`.
		runes := []rune(strings.ToLower(p))
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[i] = string(runes)
	}
	out := strings.Join(parts, "")
	if out == "" {
		return pyName
	}
	if !isExportableLetter(rune(out[0])) {
		return "Tag" + out
	}
	// Append a stable suffix when the cleaned name doesn't already
	// end with `Tag` / `Format` / `Scope` to keep the type role
	// obvious in Go code (e.g. `SystemTag`, `FontRunTag`).
	switch {
	case out == "HeapFormat", out == "NodeScope":
		return out
	case strings.HasSuffix(out, "Tag"):
		return out
	default:
		return out + "Tag"
	}
}

func isExportableLetter(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// goNameFor cleans up a pylabview enum-member name for Go. It strips
// `SL__` / `OF__` prefixes and capitalises the first character. The
// remaining name keeps its mixed-case form so labels like
// `activeDiag` survive as `ActiveDiag`.
func goNameFor(pyName string) string {
	s := strings.TrimPrefix(pyName, "SL__")
	s = strings.TrimPrefix(s, "OF__")
	if s == "" {
		return pyName
	}
	first := strings.ToUpper(string(s[0]))
	return first + s[1:]
}

func parseLVheap(r io.Reader) ([]*enumSpec, error) {
	var (
		out []*enumSpec
		cur *enumSpec
	)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if m := classRE.FindStringSubmatch(line); m != nil {
			pyName, base := m[1], m[2]
			if !enumBaseClasses[base] {
				cur = nil
				continue
			}
			cur = &enumSpec{
				pyName: pyName,
				goType: goTypeFor(pyName),
				seen:   map[string]int{},
			}
			out = append(out, cur)
			continue
		}
		if cur == nil {
			continue
		}
		if m := memberRE.FindStringSubmatch(line); m != nil {
			name := m[1]
			val, err := strconv.Atoi(m[2])
			if err != nil {
				return nil, fmt.Errorf("parse %s.%s: %w", cur.pyName, name, err)
			}
			if _, dup := cur.seen[name]; dup {
				return nil, fmt.Errorf("%s: duplicate member %s", cur.pyName, name)
			}
			cur.seen[name] = val
			cur.members = append(cur.members, enumMember{
				pyName: name,
				goName: goNameFor(name),
				value:  val,
			})
			continue
		}
		// blank lines / docstrings / methods inside the class are
		// ignored; once we hit `def ...` or unindented content we
		// effectively keep adding to cur until the next `class` line
		// resets it. That's fine for LVheap.py: enum classes don't
		// declare methods of their own.
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// goTypeNames keep track of name collisions across enums so we can
// disambiguate when two pylabview names map to the same Go type.
func resolveTypeCollisions(specs []*enumSpec) {
	seen := map[string][]*enumSpec{}
	for _, s := range specs {
		seen[s.goType] = append(seen[s.goType], s)
	}
	for typeName, list := range seen {
		if len(list) <= 1 {
			continue
		}
		// Disambiguate by appending the original pyName (without
		// underscores) for every collision after the first.
		for i, s := range list {
			if i == 0 {
				continue
			}
			s.goType = typeName + "From" + strings.ReplaceAll(s.pyName, "_", "")
		}
	}
}

// disambiguateGoNames walks the spec's members and rewrites collisions
// in goName so each member emits a unique Go constant identifier. The
// first occurrence keeps its name; subsequent ones get a numeric suffix
// (`Foo`, `Foo2`, `Foo3`, …). pylabview occasionally declares two
// members differing only by the case of the first letter after the
// prefix (e.g. `OF__commentSelLabData` and `OF__CommentSelLabData`),
// which collapse to the same `goNameFor` output.
func disambiguateGoNames(specs []*enumSpec) {
	for _, s := range specs {
		used := map[string]int{}
		for i := range s.members {
			name := s.members[i].goName
			n := used[name]
			if n > 0 {
				s.members[i].goName = fmt.Sprintf("%s%d", name, n+1)
			}
			used[name] = n + 1
		}
	}
}

func emit(out io.Writer, specs []*enumSpec) {
	fmt.Fprintf(out, "// Code generated by scripts/gen-heap-tags. DO NOT EDIT.\n")
	fmt.Fprintf(out, "// Source: references/pylabview/pylabview/LVheap.py\n")
	fmt.Fprintf(out, "// Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(out, "package heap\n\n")
	fmt.Fprintf(out, "import \"fmt\"\n\n")

	for _, s := range specs {
		fmt.Fprintf(out, "// %s mirrors pylabview's %s enum.\n", s.goType, s.pyName)
		fmt.Fprintf(out, "type %s int\n\n", s.goType)
		if len(s.members) == 0 {
			fmt.Fprintf(out, "// %s has no documented members in pylabview.\n\n", s.goType)
			continue
		}
		// Constants in declaration order so aliases (same value, different
		// names) preserve pylabview's source ordering.
		fmt.Fprintf(out, "// Members of %s.\n", s.goType)
		fmt.Fprintf(out, "const (\n")
		for _, m := range s.members {
			constName := s.goType + m.goName
			fmt.Fprintf(out, "\t%s %s = %d // %s\n", constName, s.goType, m.value, m.pyName)
		}
		fmt.Fprintf(out, ")\n\n")

		// Names map: first-declared name wins per value (matches
		// pylabview's `_value2member_map_` lookup behaviour where
		// aliases resolve to the canonical first member).
		seenValue := map[int]bool{}
		fmt.Fprintf(out, "var %sNames = map[%s]string{\n", lowerFirst(s.goType), s.goType)
		// Stable order by value for readable output.
		sorted := append([]enumMember(nil), s.members...)
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i].value != sorted[j].value {
				return sorted[i].value < sorted[j].value
			}
			return sorted[i].goName < sorted[j].goName
		})
		// First pass over the *declaration* order to pick canonical names.
		canonical := map[int]string{}
		for _, m := range s.members {
			if _, ok := canonical[m.value]; !ok {
				canonical[m.value] = m.pyName
			}
		}
		for _, m := range sorted {
			if seenValue[m.value] {
				continue
			}
			seenValue[m.value] = true
			constName := s.goType + m.goName
			fmt.Fprintf(out, "\t%s: %q,\n", constName, canonical[m.value])
		}
		fmt.Fprintf(out, "}\n\n")

		fmt.Fprintf(out, "// String returns the pylabview name when known, otherwise a `%s(N)` fallback.\n", s.goType)
		fmt.Fprintf(out, "func (t %s) String() string {\n", s.goType)
		fmt.Fprintf(out, "\tif name, ok := %sNames[t]; ok {\n\t\treturn name\n\t}\n", lowerFirst(s.goType))
		fmt.Fprintf(out, "\treturn fmt.Sprintf(%q, int(t))\n", s.goType+"(%d)")
		fmt.Fprintf(out, "}\n\n")
	}
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(string(s[0])) + s[1:]
}

func main() {
	in, err := os.Open("references/pylabview/pylabview/LVheap.py")
	if err != nil {
		fmt.Fprintln(os.Stderr, "open LVheap.py:", err)
		os.Exit(1)
	}
	defer in.Close()
	specs, err := parseLVheap(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}
	resolveTypeCollisions(specs)
	disambiguateGoNames(specs)
	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, "no enums extracted")
		os.Exit(1)
	}
	emit(os.Stdout, specs)
	fmt.Fprintln(os.Stderr, "extracted", len(specs), "enums")
}
