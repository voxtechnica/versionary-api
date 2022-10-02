package device

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	ua "github.com/voxtechnica/user-agent"
	v "github.com/voxtechnica/versionary"
)

//==============================================================================
// Device Table
//==============================================================================

// rowDevices is a TableRow definition for Device versions.
var rowDevices = v.TableRow[Device]{
	RowName:      "devices_version",
	PartKeyName:  "id",
	PartKeyValue: func(d Device) string { return d.ID },
	SortKeyName:  "update_id",
	SortKeyValue: func(d Device) string { return d.VersionID },
	JsonValue:    func(d Device) []byte { return d.CompressedJSON() },
	TimeToLive:   func(d Device) int64 { return d.ExpiresAt.Unix() },
}

// rowDevicesUser is a TableRow definition for current Device versions,
// partitioned by UserID.
var rowDevicesUser = v.TableRow[Device]{
	RowName:      "devices_user",
	PartKeyName:  "user_id",
	PartKeyValue: func(d Device) string { return d.UserID },
	SortKeyName:  "id",
	SortKeyValue: func(d Device) string { return d.ID },
	JsonValue:    func(d Device) []byte { return d.CompressedJSON() },
	TimeToLive:   func(d Device) int64 { return d.ExpiresAt.Unix() },
}

// rowDevicesDate is a TableRow definition for current Device versions,
// partitioned by LastSeenOn date.
var rowDevicesDate = v.TableRow[Device]{
	RowName:      "devices_date",
	PartKeyName:  "date",
	PartKeyValue: func(d Device) string { return d.LastSeenOn() },
	SortKeyName:  "id",
	SortKeyValue: func(d Device) string { return d.ID },
	JsonValue:    func(d Device) []byte { return d.CompressedJSON() },
	TimeToLive:   func(d Device) int64 { return d.ExpiresAt.Unix() },
}

// NewDeviceTable instantiates a new DynamoDB Device table.
func NewDeviceTable(dbClient *dynamodb.Client, env string) v.Table[Device] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Device]{
		Client:     dbClient,
		EntityType: "Device",
		TableName:  "devices" + "_" + env,
		TTL:        true,
		EntityRow:  rowDevices,
		IndexRows: map[string]v.TableRow[Device]{
			rowDevicesDate.RowName: rowDevicesDate,
			rowDevicesUser.RowName: rowDevicesUser,
		},
	}
}

// NewDeviceMemTable creates an in-memory Device table for testing purposes.
func NewDeviceMemTable(table v.Table[Device]) v.MemTable[Device] {
	return v.NewMemTable(table)
}

//==============================================================================
// Device Service
//==============================================================================

// DeviceService is used to manage a Device database.
type DeviceService struct {
	EntityType string
	Table      v.TableReadWriter[Device]
}

//------------------------------------------------------------------------------
// Device Versions
//------------------------------------------------------------------------------

// Create a Device in the Device table.
func (s DeviceService) Create(ctx context.Context, header, userID string) (Device, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	d := Device{
		ID:         t.String(),
		CreatedAt:  at,
		VersionID:  t.String(),
		UpdatedAt:  at,
		LastSeenAt: at,
		ExpiresAt:  at.AddDate(1, 0, 0),
		UserID:     userID,
		UserAgent:  ua.Parse(header),
	}
	problems := d.Validate()
	if len(problems) > 0 {
		return d, problems, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, d.ID, strings.Join(problems, ", "))
	}
	err := s.Table.WriteEntity(ctx, d)
	if err != nil {
		return d, problems, fmt.Errorf("error creating %s %s: %w", s.EntityType, d.ID, err)
	}
	return d, problems, nil
}

// Update a Device in the Device table. If the UserAgent has changed, the Device will get a new VersionID.
// Otherwise, the VersionID will remain the same, and only the LastSeenAt and ExpiresAt times will be updated.
// Note that we're writing (not updating) the Device so that historical Devices by Date are preserved.
func (s DeviceService) Update(ctx context.Context, deviceID, header, userID string) (Device, []string, error) {
	if deviceID == "" || !tuid.IsValid(tuid.TUID(deviceID)) {
		return Device{}, []string{}, errors.New("error updating Device: a valid deviceID is required")
	}
	if userID != "" && !tuid.IsValid(tuid.TUID(userID)) {
		return Device{}, []string{}, errors.New("error updating Device: the provided userID is invalid")
	}
	d, err := s.Table.ReadEntity(ctx, deviceID)
	// Create a new Device if the provided ID is not found. It may be very old, and the TTL may have expired.
	if err != nil && errors.Is(err, v.ErrNotFound) {
		return s.Create(ctx, header, userID)
	}
	if err != nil {
		return d, nil, fmt.Errorf("error updating %s %s: %w", s.EntityType, deviceID, err)
	}
	t := tuid.NewID()
	at, _ := t.Time()
	// If the UserAgent has changed, create a new Device version.
	if d.UserAgent.Header != header {
		d.UserAgent = ua.Parse(header)
		d.VersionID = t.String()
		d.UpdatedAt = at
	}
	// Refresh the LastSeenAt and ExpiresAt times.
	d.LastSeenAt = at
	d.ExpiresAt = at.AddDate(1, 0, 0)
	problems := d.Validate()
	if len(problems) > 0 {
		return d, problems, fmt.Errorf("error updating %s %s: invalid field(s): %s", s.EntityType, d.ID, strings.Join(problems, ", "))
	}
	// Write (not Update) the Device so that historical Devices by Date are preserved.
	return d, problems, s.Table.WriteEntity(ctx, d)
}

// Write a Device to the Device table. This method assumes that the Device has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Device table.
func (s DeviceService) Write(ctx context.Context, o Device) (Device, error) {
	return o, s.Table.WriteEntity(ctx, o)
}

// Delete a Device from the Device table. The deleted Device is returned.
func (s DeviceService) Delete(ctx context.Context, id string) (Device, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Exists checks if a Device exists in the Device table.
func (s DeviceService) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// Read a specified Device from the Device table.
func (s DeviceService) Read(ctx context.Context, id string) (Device, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Device from the Device table, serialized as JSON.
func (s DeviceService) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// VersionExists checks if a specified Device version exists in the Device table.
func (s DeviceService) VersionExists(ctx context.Context, id, versionID string) bool {
	return s.Table.EntityVersionExists(ctx, id, versionID)
}

// ReadVersion gets a specified Device version from the Device table.
func (s DeviceService) ReadVersion(ctx context.Context, id, versionID string) (Device, error) {
	return s.Table.ReadEntityVersion(ctx, id, versionID)
}

// ReadVersionAsJSON gets a specified Device version from the Device table, serialized as JSON.
func (s DeviceService) ReadVersionAsJSON(ctx context.Context, id, versionID string) ([]byte, error) {
	return s.Table.ReadEntityVersionAsJSON(ctx, id, versionID)
}

// ReadVersions returns paginated versions of the specified Device.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s DeviceService) ReadVersions(ctx context.Context, id string, reverse bool, limit int, offset string) ([]Device, error) {
	return s.Table.ReadEntityVersions(ctx, id, reverse, limit, offset)
}

// ReadVersionsAsJSON returns paginated versions of the specified Device, serialized as JSON.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s DeviceService) ReadVersionsAsJSON(ctx context.Context, id string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntityVersionsAsJSON(ctx, id, reverse, limit, offset)
}

// ReadAllVersions returns all versions of the specified Device in chronological order.
// Caution: this may be a LOT of data!
func (s DeviceService) ReadAllVersions(ctx context.Context, id string) ([]Device, error) {
	return s.Table.ReadAllEntityVersions(ctx, id)
}

// ReadAllVersionsAsJSON returns all versions of the specified Device, serialized as JSON.
// Caution: this may be a LOT of data!
func (s DeviceService) ReadAllVersionsAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadAllEntityVersionsAsJSON(ctx, id)
}

// ReadDeviceIDs returns a paginated list of Device IDs in the Device table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s DeviceService) ReadDeviceIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadDevices returns a paginated list of Devices in the Device table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Devices, retrieved individually, in parallel.
// It is probably not the best way to page through a large Device table.
func (s DeviceService) ReadDevices(ctx context.Context, reverse bool, limit int, offset string) []Device {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Device{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//------------------------------------------------------------------------------
// Devices by UserID
//------------------------------------------------------------------------------

// ReadUserIDs returns a paginated UserID list for which there are Devices in the Device table.
// Sorting is alphabetical (or reverse). The offset is the last UserID returned in a previous request.
func (s DeviceService) ReadUserIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowDevicesUser, reverse, limit, offset)
}

// ReadAllUserIDs returns a complete, alphabetical UserID list for which there are Devices in the Device table.
func (s DeviceService) ReadAllUserIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowDevicesUser)
}

// ReadDevicesByUserID returns paginated Devices by UserID. Sorting is chronological (or reverse).
// The offset is the ID of the last Device returned in a previous request.
func (s DeviceService) ReadDevicesByUserID(ctx context.Context, userID string, reverse bool, limit int, offset string) ([]Device, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowDevicesUser, userID, reverse, limit, offset)
}

// ReadDevicesByUserIDAsJSON returns paginated JSON Devices by UserID. Sorting is chronological (or reverse).
// The offset is the ID of the last Device returned in a previous request.
func (s DeviceService) ReadDevicesByUserIDAsJSON(ctx context.Context, userID string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowDevicesUser, userID, reverse, limit, offset)
}

// ReadAllDevicesByUserID returns the complete list of Devices, sorted chronologically by CreatedAt timestamp.
func (s DeviceService) ReadAllDevicesByUserID(ctx context.Context, userID string) ([]Device, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowDevicesUser, userID)
}

// ReadAllDevicesByUserIDAsJSON returns the complete list of Devices, serialized as JSON.
func (s DeviceService) ReadAllDevicesByUserIDAsJSON(ctx context.Context, userID string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowDevicesUser, userID)
}

//------------------------------------------------------------------------------
// Devices by Date (YYYY-MM-DD)
//------------------------------------------------------------------------------

// ReadDates returns a paginated Date list for which there are Devices in the Device table.
// Sorting is chronological (or reverse). The offset is the last Date returned in a previous request.
func (s DeviceService) ReadDates(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowDevicesDate, reverse, limit, offset)
}

// ReadAllDates returns a complete, chronological list of Dates for which there are Devices in the Device table.
func (s DeviceService) ReadAllDates(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowDevicesDate)
}

// ReadDevicesByDate returns paginated Devices by Date. Sorting is chronological (or reverse).
// The offset is the ID of the last Device returned in a previous request.
func (s DeviceService) ReadDevicesByDate(ctx context.Context, date string, reverse bool, limit int, offset string) ([]Device, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowDevicesDate, date, reverse, limit, offset)
}

// ReadDevicesByDateAsJSON returns paginated JSON Devices by Date. Sorting is chronological (or reverse).
// The offset is the ID of the last Device returned in a previous request.
func (s DeviceService) ReadDevicesByDateAsJSON(ctx context.Context, date string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowDevicesDate, date, reverse, limit, offset)
}

// ReadAllDevicesByDate returns the complete list of Devices, sorted chronologically by CreatedAt timestamp.
func (s DeviceService) ReadAllDevicesByDate(ctx context.Context, date string) ([]Device, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowDevicesDate, date)
}

// ReadAllDevicesByDateAsJSON returns the complete list of Devices, serialized as JSON.
func (s DeviceService) ReadAllDevicesByDateAsJSON(ctx context.Context, date string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowDevicesDate, date)
}

// CountDevicesByDate returns a DeviceCount for Devices in the Device table on the specified Date.
func (s DeviceService) CountDevicesByDate(ctx context.Context, date string) (Count, error) {
	dc := Count{}
	if date == "" || !dateRegex.MatchString(date) {
		return dc, fmt.Errorf("count devices by date: invalid date: %s", date)

	}
	dc.Date = date
	limit := 10000
	offset := "-"
	devices, err := s.ReadDevicesByDate(ctx, date, false, limit, offset)
	if err != nil {
		return dc, fmt.Errorf("count devices by date %s: %w", date, err)
	}
	for len(devices) > 0 {
		for _, d := range devices {
			dc = dc.Increment(d)
		}
		offset = devices[len(devices)-1].ID
		devices, err = s.ReadDevicesByDate(ctx, date, false, limit, offset)
		if err != nil {
			return dc, fmt.Errorf("count devices by date %s: %w", date, err)
		}
	}
	return dc, nil
}
