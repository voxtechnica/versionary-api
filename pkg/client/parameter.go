package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// ErrParameterNotFound is returned when a parameter is not found in the Parameter Store.
var ErrParameterNotFound = errors.New("parameter store: parameter not found")

// Parameter represents a simplified AWS parameter in the SSM Parameter Store.
type Parameter struct {
	// Name is the case-sensitive name of the parameter. The maximum usable length is 1011 characters.
	// Slashes can be used to create a hierarchy of grouped parameters and support "starts with" searches.
	// For example: /AppName/3rdPartyName/ParameterName
	Name string `json:"name"`

	// Value is the value of the parameter. The size limit is 4096 bytes.
	Value string `json:"value"`
}

// ParameterStore is a caching client for AWS SSM Parameter Store. If the AWS SSM Client is not configured,
// the cache is still used as an in-memory parameter store, useful for testing purposes.
type ParameterStore struct {
	Client *ssm.Client
	Cache  *map[string]Parameter
}

// SetParameter sets the provided parameter in the cache and updates the AWS SSM Parameter Store.
func (ps ParameterStore) SetParameter(ctx context.Context, p Parameter) error {
	if p.Name == "" {
		return errors.New("parameter store: missing parameter name")
	}
	// Update the cache.
	if ps.Cache == nil {
		ps.Cache = &map[string]Parameter{}
	}
	(*ps.Cache)[p.Name] = p
	// If the client is not set, the parameter store is in-memory only.
	if ps.Client == nil {
		return nil
	}
	// If the client is set, the parameter store is backed by AWS SSM.
	input := &ssm.PutParameterInput{
		Name:      &p.Name,
		Value:     &p.Value,
		Type:      types.ParameterTypeSecureString,
		Overwrite: aws.Bool(true),
	}
	_, err := ps.Client.PutParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("parameter store: set parameter %s: %w", p.Name, err)
	}
	return nil
}

// GetParameter returns the provided parameter with the matching value.
func (ps ParameterStore) GetParameter(ctx context.Context, p Parameter) (Parameter, error) {
	if p.Name == "" {
		return p, errors.New("parameter store: missing parameter name")
	}
	if ps.Cache == nil && ps.Client == nil {
		return p, ErrParameterNotFound
	}
	if ps.Cache == nil {
		ps.Cache = &map[string]Parameter{}
	}
	// Try the cache first.
	cached, ok := (*ps.Cache)[p.Name]
	if ok {
		return cached, nil
	}
	// If the client is not set, the parameter store is in-memory only.
	if ps.Client == nil {
		return p, ErrParameterNotFound
	}
	// If the client is set, the parameter store is backed by AWS SSM.
	input := &ssm.GetParameterInput{
		Name:           &p.Name,
		WithDecryption: aws.Bool(true), // ignored if the parameter is not encrypted
	}
	result, err := ps.Client.GetParameter(ctx, input)
	if err != nil {
		var nf *types.ParameterNotFound
		if errors.As(err, &nf) {
			return p, ErrParameterNotFound
		}
		return p, fmt.Errorf("parameter store: get parameter %s: %w", p.Name, err)
	}
	p.Value = *result.Parameter.Value
	// Cache the parameter.
	(*ps.Cache)[p.Name] = p
	return p, nil
}

// GetParameters returns the provided parameters, in order, with their values populated.
func (ps ParameterStore) GetParameters(ctx context.Context, params []Parameter) ([]Parameter, error) {
	if len(params) == 0 {
		return nil, nil
	}
	if ps.Cache == nil && ps.Client == nil {
		return nil, ErrParameterNotFound
	}
	if ps.Cache == nil {
		ps.Cache = &map[string]Parameter{}
	}
	// Try the cache first.
	var missing []string
	for _, p := range params {
		c, ok := (*ps.Cache)[p.Name]
		if ok {
			p.Value = c.Value
		} else {
			missing = append(missing, p.Name)
		}
	}
	if len(missing) == 0 {
		return params, nil
	}
	// If the client is not set, the parameter store is in-memory only.
	if ps.Client == nil {
		return params, ErrParameterNotFound
	}
	// If the client is set, the parameter store is backed by AWS SSM.
	input := &ssm.GetParametersInput{
		Names:          missing,        // fetch all missing parameters
		WithDecryption: aws.Bool(true), // ignored if the parameter is not encrypted
	}
	result, err := ps.Client.GetParameters(ctx, input)
	if err != nil {
		var nf *types.ParameterNotFound
		if errors.As(err, &nf) {
			return params, ErrParameterNotFound
		}
		return params, fmt.Errorf("parameter store: get parameters %v: %w", missing, err)
	}
	// Cache the parameters.
	for _, p := range result.Parameters {
		(*ps.Cache)[*p.Name] = Parameter{
			Name:  *p.Name,
			Value: *p.Value,
		}
	}
	// Update the parameter values.
	missing = missing[:0]
	for i, p := range params {
		c, ok := (*ps.Cache)[p.Name]
		if ok {
			params[i].Value = c.Value
		} else {
			missing = append(missing, p.Name)
		}
	}
	if len(missing) > 0 {
		return params, ErrParameterNotFound
	}
	return params, nil
}

// DeleteParameter deletes the provided parameter from the cache and the AWS SSM Parameter Store.
func (ps ParameterStore) DeleteParameter(ctx context.Context, p Parameter) error {
	if p.Name == "" {
		return errors.New("parameter store: missing parameter name")
	}
	// Delete from the cache.
	if ps.Cache != nil {
		delete(*ps.Cache, p.Name)
	}
	// If the client is not set, the parameter store is in-memory only.
	if ps.Client == nil {
		return nil
	}
	// If the client is set, the parameter store is backed by AWS SSM.
	input := &ssm.DeleteParameterInput{
		Name: &p.Name,
	}
	_, err := ps.Client.DeleteParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("parameter store: delete parameter %s: %w", p.Name, err)
	}
	return nil
}

// NewParameterStore returns a new caching ParameterStore, backed by AWS SSM Parameter Store.
func NewParameterStore(cfg aws.Config) ParameterStore {
	return ParameterStore{
		Client: ssm.NewFromConfig(cfg),
		Cache:  &map[string]Parameter{},
	}
}

// NewParameterStoreMock returns a new mock parameter store, with in-memory parameter caching.
func NewParameterStoreMock() ParameterStore {
	return ParameterStore{
		Cache: &map[string]Parameter{},
	}
}
