package domain

type ListQuery struct {
	PageSize            int
	Cursor              string
	Scoped              bool
	Global              bool
	Warehouses          []string
	ActorID             string
	AuthorizationDigest string
}
type ProfilePage struct {
	Items      []Profile `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}
