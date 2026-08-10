package domain

// Role is a closed enum of the roles a User may hold, per
// docs/agent-operating-handbook.md A1's Identity/Users bounded-context
// summary. A single User can hold more than one Role at once (e.g. a Host
// who is also a Player) — see User.Roles, a slice, not a single value.
type Role string

const (
	RolePlayer        Role = "player"
	RoleHostOrganiser Role = "host_organiser"
	RoleGameAdmin     Role = "game_admin"
	RoleFacilityOwner Role = "facility_owner"
	RoleClub          Role = "club"
	RolePlatformAdmin Role = "platform_admin"
)

// IsValid reports whether r is one of the six closed-enum Role values above.
// The empty string (the zero value) is never valid — a User with an
// unset/garbage Role must be rejected by NewUser, not silently accepted.
func (r Role) IsValid() bool {
	switch r {
	case RolePlayer, RoleHostOrganiser, RoleGameAdmin, RoleFacilityOwner, RoleClub, RolePlatformAdmin:
		return true
	default:
		return false
	}
}
