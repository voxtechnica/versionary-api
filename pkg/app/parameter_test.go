package app

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	"testing"
)

// TestParameterStoreCRUD tests the CRUD operations of the ParameterStore.
// This is an integration test that requires AWS credentials to be set in the environment.
func TestParameterStoreCRUD(t *testing.T) {
	expect := assert.New(t)

	// Create a new parameter store.
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatalf("error loading AWS config: %s", err)
	}
	ps := NewParameterStore(cfg)

	// Instantiate new, unique test parameters.
	t1 := tuid.NewID().String()
	p1 := Parameter{
		Name:  "/versionary/test/" + t1,
		Value: t1,
	}
	t2 := tuid.NewID().String()
	p2 := Parameter{
		Name:  "/versionary/test/" + t2,
		Value: t2,
	}

	// Create both parameters.
	err = ps.SetParameter(ctx, p1)
	if err != nil {
		t.Fatalf("failed to set parameter 1 %s: %s", p1.Name, err)
	}
	err = ps.SetParameter(ctx, p2)
	if err != nil {
		t.Fatalf("failed to set parameter 2 %s: %s", p2.Name, err)
	}

	// Read the first (cached) parameter.
	got, err := ps.GetParameter(ctx, Parameter{Name: p1.Name})
	if expect.NoError(err, "failed to get cached parameter 1 %s", p1.Name) {
		expect.Equal(p1, got, "read cached parameter 1")
	}

	// Clear the cache and try again.
	ps.Cache = &map[string]Parameter{}
	got, err = ps.GetParameter(ctx, Parameter{Name: p1.Name})
	if expect.NoError(err, "failed to get parameter 1 %s", p1.Name) {
		expect.Equal(p1, got, "read parameter 1")
	}

	// Read a non-existent parameter.
	_, err = ps.GetParameter(ctx, Parameter{Name: "/versionary/test/does-not-exist"})
	expect.ErrorIs(err, ErrParameterNotFound, "failed to get non-existent parameter")

	// Read both parameters.
	params, err := ps.GetParameters(ctx, []Parameter{
		{Name: p1.Name},
		{Name: p2.Name},
	})
	if expect.NoError(err, "failed to get parameters") {
		expect.Equal([]Parameter{p1, p2}, params, "read parameters")
	}

	// Update the second parameter.
	p2.Value = tuid.NewID().String()
	err = ps.SetParameter(ctx, p2)
	if expect.NoError(err, "failed to set parameter 2 %s", p2.Name) {
		// Clear the cache and read the updated parameter.
		ps.Cache = &map[string]Parameter{}
		got, err = ps.GetParameter(ctx, p2)
		if expect.NoError(err, "failed to get parameter 2 %s", p2.Name) {
			expect.Equal(p2, got, "read parameter 2")
		}
	}

	// Delete both parameters.
	err = ps.DeleteParameter(ctx, p1)
	if expect.NoError(err, "failed to delete parameter 1 %s", p1.Name) {
		_, err = ps.GetParameter(ctx, p1)
		expect.ErrorIs(err, ErrParameterNotFound, "failed to get deleted parameter 1")
	}
	err = ps.DeleteParameter(ctx, p2)
	if expect.NoError(err, "failed to delete parameter 2 %s", p2.Name) {
		_, err = ps.GetParameter(ctx, p2)
		expect.ErrorIs(err, ErrParameterNotFound, "failed to get deleted parameter 2")
	}
}
