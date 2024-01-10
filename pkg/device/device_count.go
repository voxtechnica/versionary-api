package device

import (
	v "github.com/voxtechnica/versionary"
)

type Count struct {
	Date        string         `json:"date"` // YYYY-MM-DD
	Total       int            `json:"total"`
	ClientTypes map[string]int `json:"clientTypes,omitempty"`
	ClientNames map[string]int `json:"clientNames,omitempty"`
	DeviceTypes map[string]int `json:"deviceTypes,omitempty"`
	OSNames     map[string]int `json:"osNames,omitempty"`
}

// Type returns the entity type of the DeviceCount.
func (c Count) Type() string {
	return "DeviceCount"
}

// CompressedJSON returns a compressed JSON representation of the DeviceCount.
func (c Count) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(c)
	if err != nil {
		return nil
	}
	return j
}

// Validate checks whether the DeviceCount has all required fields and whether
// the supplied values are valid, returning a list of problems. If the list is
// empty, then the DeviceCount is valid.
func (c Count) Validate() []string {
	problems := []string{}
	if c.Date == "" || !v.IsValidDate(c.Date) {
		problems = append(problems, "Date is missing or invalid")
	}
	return problems
}

// Increment increments the DeviceCount for the supplied Device.
func (c Count) Increment(d Device) Count {
	if c.Date == "" {
		c.Date = d.LastSeenOn()
	}
	c.Total++
	if d.UserAgent.ClientType != "" {
		c.ClientTypes[d.UserAgent.ClientType]++
	}
	if d.UserAgent.ClientName != "" {
		c.ClientNames[d.UserAgent.ClientName]++
	}
	if d.UserAgent.DeviceType != "" {
		c.DeviceTypes[d.UserAgent.DeviceType]++
	}
	if d.UserAgent.OSName != "" {
		c.OSNames[d.UserAgent.OSName]++
	}
	return c
}

// Merge merges that DeviceCount into this DeviceCount.
func (c Count) Merge(that Count) Count {
	c.Total += that.Total
	for clientType, count := range that.ClientTypes {
		c.ClientTypes[clientType] += count
	}
	for clientName, count := range that.ClientNames {
		c.ClientNames[clientName] += count
	}
	for deviceType, count := range that.DeviceTypes {
		c.DeviceTypes[deviceType] += count
	}
	for osName, count := range that.OSNames {
		c.OSNames[osName] += count
	}
	return c
}
