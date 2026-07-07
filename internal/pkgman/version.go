package pkgman

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a simple semver triple (no prerelease/build metadata).
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses "v1.2.3" or "1.2.3".
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return Version{}, fmt.Errorf("empty version")
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return Version{}, fmt.Errorf("invalid version %q", s)
	}
	parsePart := func(p string) (int, error) {
		if p == "" {
			return 0, nil
		}
		return strconv.Atoi(p)
	}
	maj, err := parsePart(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid version %q", s)
	}
	min, pat := 0, 0
	if len(parts) > 1 {
		if min, err = parsePart(parts[1]); err != nil {
			return Version{}, fmt.Errorf("invalid version %q", s)
		}
	}
	if len(parts) > 2 {
		if pat, err = parsePart(parts[2]); err != nil {
			return Version{}, fmt.Errorf("invalid version %q", s)
		}
	}
	return Version{Major: maj, Minor: min, Patch: pat}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Compare(o Version) int {
	if v.Major != o.Major {
		return v.Major - o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor - o.Minor
	}
	return v.Patch - o.Patch
}

// SatisfiesConstraint reports whether resolved meets the manifest constraint.
// Empty constraint or "*" matches any version.
func SatisfiesConstraint(constraint, resolved string) error {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return nil
	}
	rv, err := ParseVersion(normalizeVersionLabel(resolved))
	if err != nil {
		return fmt.Errorf("resolved version %q: %w", resolved, err)
	}
	switch {
	case strings.HasPrefix(constraint, ">="):
		min, err := ParseVersion(normalizeVersionLabel(strings.TrimSpace(constraint[2:])))
		if err != nil {
			return fmt.Errorf("constraint %q: %w", constraint, err)
		}
		if rv.Compare(min) < 0 {
			return fmt.Errorf("resolved %s does not satisfy %s", rv, constraint)
		}
		return nil
	case strings.HasPrefix(constraint, "^"):
		base, err := ParseVersion(normalizeVersionLabel(strings.TrimSpace(constraint[1:])))
		if err != nil {
			return fmt.Errorf("constraint %q: %w", constraint, err)
		}
		if rv.Major != base.Major || rv.Compare(base) < 0 {
			return fmt.Errorf("resolved %s does not satisfy %s", rv, constraint)
		}
		return nil
	default:
		want, err := ParseVersion(normalizeVersionLabel(constraint))
		if err != nil {
			return fmt.Errorf("constraint %q: %w", constraint, err)
		}
		if rv.Compare(want) != 0 {
			return fmt.Errorf("resolved %s does not satisfy %s", rv, constraint)
		}
		return nil
	}
}

// ResolvedVersion extracts a version label from a dependency source.
func ResolvedVersion(source string) string {
	if strings.HasPrefix(source, "git+") {
		_, ref := splitGitURL(strings.TrimPrefix(source, "git+"))
		if ref != "" {
			return normalizeVersionLabel(ref)
		}
	}
	return "0.0.0"
}

func normalizeVersionLabel(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	return s
}
