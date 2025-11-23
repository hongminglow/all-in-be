package models

type Permission struct {
	ID                    int64  `json:"id"`
	PermissionName        string `json:"name"`
	PermissionDescription string `json:"description"`
}
