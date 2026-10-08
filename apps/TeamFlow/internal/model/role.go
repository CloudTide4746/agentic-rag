package model

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// IsValidRole 判断成员角色是否合法。
func IsValidRole(role string) bool {
	return role == RoleOwner || role == RoleAdmin || role == RoleMember
}

// HasAdminPermission 判断角色是否可以管理普通成员。
func HasAdminPermission(role string) bool {
	return role == RoleOwner || role == RoleAdmin
}

// RoleAtLeast 判断实际角色是否达到要求的角色等级。
func RoleAtLeast(actual, required string) bool {
	weight := map[string]int{
		RoleOwner:  3,
		RoleAdmin:  2,
		RoleMember: 1,
	}
	return weight[actual] >= weight[required]
}
