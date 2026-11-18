package domain

type ListQuery struct {
	PageSize int
	Cursor   string
}
type ProfilePage struct {
	Items      []Profile `json:"items"`
	NextCursor *string   `json:"next_cursor"`
}
