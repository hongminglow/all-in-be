package models

const (
	NormalUser = "player"
	VIPUser    = "vip-player"
	VVIPUser   = "vvip-player"
)

type Role struct {
	ID              int64   `json:"id"`
	RoleName        string  `json:"role"`
	RoleDescription string  `json:"description"`
	Permission      []int64 `json:"permission"`
}
