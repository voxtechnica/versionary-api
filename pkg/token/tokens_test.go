package token

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

var (
	// Token Service
	ctx     = context.Background()
	service = NewMockService("test")

	// Known timestamps
	t1 = time.Date(2022, time.April, 1, 12, 0, 0, 0, time.UTC)
	t2 = time.Date(2022, time.April, 1, 13, 0, 0, 0, time.UTC)
	t3 = time.Date(2022, time.April, 1, 14, 0, 0, 0, time.UTC)
	t4 = time.Date(2022, time.April, 2, 12, 0, 0, 0, time.UTC)
	t5 = time.Date(2022, time.April, 3, 12, 0, 0, 0, time.UTC)

	// Token IDs
	id1 = tuid.NewIDWithTime(t1).String()
	id2 = tuid.NewIDWithTime(t2).String()
	id3 = tuid.NewIDWithTime(t3).String()
	id4 = tuid.NewIDWithTime(t4).String()
	id5 = tuid.NewIDWithTime(t5).String()

	// User IDs
	user1 = tuid.NewIDWithTime(t1).String()
	user2 = tuid.NewIDWithTime(t2).String()
	user3 = tuid.NewIDWithTime(t3).String()
	user4 = tuid.NewIDWithTime(t4).String()
	user5 = tuid.NewIDWithTime(t5).String()

	// Known Tokens
	t10 = Token{
		ID:        id1,
		CreatedAt: t1,
		ExpiresAt: t1.AddDate(0, 0, 30),
		UserID:    user1,
	}

	t20 = Token{
		ID:        id2,
		CreatedAt: t2,
		ExpiresAt: t1.AddDate(0, 0, 30),
		UserID:    user2,
	}

	t30 = Token{
		ID:        id3,
		CreatedAt: t3,
		ExpiresAt: t1.AddDate(0, 0, 30),
		UserID:    user3,
	}

	t40 = Token{
		ID:        id4,
		CreatedAt: t4,
		ExpiresAt: t1.AddDate(0, 0, 30),
		UserID:    user4,
	}

	t50 = Token{
		ID:        id5,
		CreatedAt: t5,
		ExpiresAt: t1.AddDate(0, 0, 30),
		UserID:    user5,
	}

	knownTokens   = []Token{t10, t20, t30, t40, t50}
	knownTokenIDs = []string{id1, id2, id3, id4, id5}
)

func TestMain(m *testing.M) {
	// Check table/row definitions
	if !service.Table.IsValid() {
		log.Fatal("invalid table configuration")
	}
	// Write known tokens
	for _, e := range []Token{t10, t20, t30, t40, t50} {
		if _, err := service.Write(ctx, e); err != nil {
			log.Fatal(err)
		}
	}
	// Run tests
	m.Run()
}

func _TestTokenCRUD(t *testing.T) {
	
}

func TestTokenExists(t *testing.T) {
	expect := assert.New(t)
	// Test token exists
	expect.True(service.Exists(ctx, id1))
}

func TestReadToken(t *testing.T) {
	expect := assert.New(t)
	// Test read token
	token, err := service.Read(ctx, id1)
	if expect.NoError(err) {
		expect.Equal(t10, token)
	}
}

func TestReadAsJSON(t *testing.T) {
	expect := assert.New(t)
	// Test read token as JSON
	json, err := service.ReadAsJSON(ctx, id2)
	if expect.NoError(err) {
		expect.Contains(string(json), id2)
	}
}

func TestReadIDs(t *testing.T) {
	expect := assert.New(t)
	// Test read token IDs
	ids, err := service.ReadIDs(ctx, false, 10, "")
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(ids), 5)
	}
}

func TestReadAllIDs(t *testing.T) {
	expect := assert.New(t)
	// Test read all token IDs
	allIDs, err := service.ReadAllIDs(ctx, false)
	onlyTokenIDs := v.Map(allIDs, func(entry v.TextValue) string { return entry.Key })
	if expect.NoError(err) {
		expect.Subset(onlyTokenIDs, knownTokenIDs)
	}
}

func TestReadTokens(t *testing.T) {
	expect := assert.New(t)
	// Test read tokens
	tokens := service.ReadTokens(ctx, false, 10, "")
	if expect.NotEmpty(tokens) {
		expect.Equal(5, len(tokens))
		expect.Equal(knownTokens, tokens)
	}
}

//------------------------------------------------------------------------------
// Tokens by User ID
//------------------------------------------------------------------------------

func _TestReadUsers(t *testing.T) {

}

func _TestReadAllUsers(t *testing.T) {

}

func _TestFilterUsers(t *testing.T) {

}

func _TestReadUserIDs(t *testing.T) {

}

func _TestReadAllUserIDs(t *testing.T) {

}

func _TestReadAllTokenIDsByUserID(t *testing.T) {

}

func _TestReadTokensByUserID(t *testing.T) {

}

func _TestReadTokensByUserIDAsJSON(t *testing.T) {

}

func _TestReadAllTokensByUserID(t *testing.T) {

}

func _TestReadAllTokensByUserIDAsJSON(t *testing.T) {

}

func _TestDeleteAllTokensByUserID(t *testing.T) {

}
