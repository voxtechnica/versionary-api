package user

import (
	"time"

	v "github.com/voxtechnica/versionary"
)

type Group struct {
	ID        string    `json:"id"`
	UpdateID  string    `json:"updateId"`
	ParentID  string    `json:"parentId"`
	Name      string    `json:"name"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CompressedJSON returns a compressed JSON representation of the Group.
func (g Group) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(g)
	if err != nil {
		// skip persistence if we can't serialize and compress the event
		return nil
	}
	return j
}
