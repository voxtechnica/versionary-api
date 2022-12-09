package view

import (
	"versionary-api/pkg/util"

	v "github.com/voxtechnica/versionary"
)

type Count struct {
	Date          string         `json:"date"` // YYYY-MM-DD
	Total         int            `json:"total"`
	ClientTypes   map[string]int `json:"clientTypes,omitempty"`
	ClientNames   map[string]int `json:"clientNames,omitempty"`
	DeviceTypes   map[string]int `json:"viewTypes,omitempty"`
	OSNames       map[string]int `json:"osNames,omitempty"`
	CountryCodes  map[string]int `json:"countryCodes,omitempty"`
	DeviceIDs     map[string]int `json:"deviceIDs,omitempty"`
	PageIDs       map[string]int `json:"pageIDs,omitempty"`
	PageTypes     map[string]int `json:"pageTypes,omitempty"`
	PagePaths     map[string]int `json:"pagePaths,omitempty"`
	Referrers     map[string]int `json:"referrers,omitempty"`
	SearchEngines map[string]int `json:"searchEngines,omitempty"`
	TagKeys       map[string]int `json:"tagKeys,omitempty"`
	TagValues     map[string]int `json:"tagValues,omitempty"`
}

// Type returns the entity type of the ViewCount.
func (c Count) Type() string {
	return "ViewCount"
}

// CompressedJSON returns a compressed JSON representation of the ViewCount.
func (c Count) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(c)
	if err != nil {
		return nil
	}
	return j
}

// Validate checks whether the ViewCount has all required fields and whether
// the supplied values are valid, returning a list of problems. If the list is
// empty, then the ViewCount is valid.
func (c Count) Validate() []string {
	problems := []string{}
	if c.Date == "" || !util.IsValidDate(c.Date) {
		problems = append(problems, "Date is missing or invalid")
	}
	return problems
}

// Increment increments the ViewCount for the supplied View.
func (c Count) Increment(view View) Count {
	if c.Date == "" {
		c.Date = view.CreatedOn()
	}
	c.Total++
	if view.Client.UserAgent.ClientType != "" {
		c.ClientTypes[view.Client.UserAgent.ClientType]++
	}
	if view.Client.UserAgent.ClientName != "" {
		c.ClientNames[view.Client.UserAgent.ClientName]++
	}
	if view.Client.UserAgent.DeviceType != "" {
		c.DeviceTypes[view.Client.UserAgent.DeviceType]++
	}
	if view.Client.UserAgent.OSName != "" {
		c.OSNames[view.Client.UserAgent.OSName]++
	}
	if view.Client.CountryCode != "" {
		c.CountryCodes[view.Client.CountryCode]++
	}
	if view.Client.DeviceID != "" {
		c.DeviceIDs[view.Client.DeviceID]++
	}
	if view.Page.ID != "" {
		c.PageIDs[view.Page.ID]++
	}
	if view.Page.Type != "" {
		c.PageTypes[view.Page.Type]++
	}
	path := view.Page.Path()
	if path != "" {
		c.PagePaths[path]++
	}
	referrer := view.Page.ReferrerDomain()
	if referrer != "" {
		c.Referrers[referrer]++
	}
	searchEngine := view.Page.SearchEngine()
	if searchEngine != "" {
		c.SearchEngines[searchEngine]++
	}
	for key, value := range view.Tags {
		if key != "" && value != "" {
			c.TagKeys[key]++
			c.TagValues[key+":"+value]++
		}
	}
	return c
}

// Merge merges that ViewCount into this ViewCount.
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
	for countryCode, count := range that.CountryCodes {
		c.CountryCodes[countryCode] += count
	}
	for deviceID, count := range that.DeviceIDs {
		c.DeviceIDs[deviceID] += count
	}
	for pageID, count := range that.PageIDs {
		c.PageIDs[pageID] += count
	}
	for pageType, count := range that.PageTypes {
		c.PageTypes[pageType] += count
	}
	for pagePath, count := range that.PagePaths {
		c.PagePaths[pagePath] += count
	}
	for referrer, count := range that.Referrers {
		c.Referrers[referrer] += count
	}
	for searchEngine, count := range that.SearchEngines {
		c.SearchEngines[searchEngine] += count
	}
	for tagKey, count := range that.TagKeys {
		c.TagKeys[tagKey] += count
	}
	for tagValue, count := range that.TagValues {
		c.TagValues[tagValue] += count
	}
	return c
}
