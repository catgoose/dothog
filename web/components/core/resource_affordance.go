package components

// IdentitySize selects avatar and label density for the identity primitives.
type IdentitySize int

// Identity densities: Compact suits dense table rows, Base the default chip.
const (
	IdentitySizeCompact IdentitySize = iota
	IdentitySizeBase
)

// IdentityChip is the app-neutral identity view-model the identity primitives
// render. Every field is precomputed by the caller, so the primitives carry no
// person/user/directory domain type. Secondary always renders outside the
// linked name, so it never reads as its own link.
type IdentityChip struct {
	DisplayName  string
	Secondary    string
	AvatarURL    string
	Initials     string
	FallbackBg   string
	FallbackText string
	Resolved     bool
}

func (c IdentityChip) glyph() string {
	if c.Initials != "" {
		return c.Initials
	}
	return "?"
}

func (c IdentityChip) avatarBgClass() string {
	if c.FallbackBg != "" {
		return c.FallbackBg
	}
	return "bg-base-300"
}

func (c IdentityChip) avatarTextClass() string {
	if c.FallbackText != "" {
		return c.FallbackText
	}
	return "text-base-content"
}

func identityAvatarClass(s IdentitySize) string {
	if s == IdentitySizeCompact {
		return "w-6 h-6 rounded-xl text-[10px]"
	}
	return "w-8 h-8 rounded-xl text-xs"
}

func identityNameClass(s IdentitySize) string {
	if s == IdentitySizeCompact {
		return "text-xs font-medium"
	}
	return "text-sm font-medium"
}

func identitySecondaryClass(s IdentitySize) string {
	if s == IdentitySizeCompact {
		return "text-[10px]"
	}
	return "text-xs"
}

// textLinkClass is the at-rest entity-link class with the caller's extra
// classes appended, kept in a helper so an empty extraClass yields a clean
// `link` with no trailing space.
func textLinkClass(extraClass string) string {
	if extraClass == "" {
		return "link"
	}
	return "link " + extraClass
}

// actionLinkVariant resolves the DaisyUI variant/size for an action link: the
// caller's classes when supplied, else the neutral btn-sm. The markup emits
// only `btn`, so the caller owns the single size class and a caller-supplied
// size never collides with a hardcoded one.
func actionLinkVariant(extraClass string) string {
	if extraClass == "" {
		return "btn-sm"
	}
	return extraClass
}

// destinationLinkClass is a visible at-rest link class (never `btn`) with the
// caller's extra classes appended. Navigation reads as a link; button styling
// is reserved for actions.
func destinationLinkClass(extraClass string) string {
	const base = "link link-primary font-medium whitespace-nowrap"
	if extraClass == "" {
		return base
	}
	return base + " " + extraClass
}
