package user

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
	// User Service
	ctx     = context.Background()
	service = NewMockService("test")

	// Known timestamps
	t1 = time.Date(2022, time.April, 1, 12, 0, 0, 0, time.UTC)
	t2 = time.Date(2022, time.April, 1, 13, 0, 0, 0, time.UTC)
	t3 = time.Date(2022, time.April, 1, 14, 0, 0, 0, time.UTC)
	t4 = time.Date(2022, time.April, 2, 12, 0, 0, 0, time.UTC)
	t5 = time.Date(2022, time.April, 3, 12, 0, 0, 0, time.UTC)

	// User IDs
	id1 = tuid.NewIDWithTime(t1).String()
	id2 = tuid.NewIDWithTime(t2).String()
	id3 = tuid.NewIDWithTime(t3).String()
	id4 = tuid.NewIDWithTime(t4).String()
	id5 = tuid.NewIDWithTime(t5).String()

	// Organization IDs
	orgID1 = tuid.NewIDWithTime(t1).String()
	orgID2 = tuid.NewIDWithTime(t2).String()
	orgID3 = tuid.NewIDWithTime(t2).String()
	orgID4 = tuid.NewIDWithTime(t3).String()

	// Known Users
	u10 = User{
		ID:         id1,
		VersionID:  id1,
		CreatedAt:  t1,
		UpdatedAt:  t1,
		GivenName:  "User_1",
		FamilyName: "Family_1",
		Email:      "test_user_one@test.com",
		Status:     PENDING,
		Roles:      []string{"admin", "assistant", "manager", "owner"},
		OrgID:      orgID1,
		OrgName:    "Test Organization 1",
	}

	u20 = User{
		ID:         id2,
		VersionID:  id2,
		CreatedAt:  t2,
		UpdatedAt:  t2,
		GivenName:  "User_2",
		FamilyName: "Family_2",
		Email:      "test_user_two@test.com",
		Status:     ENABLED,
		Roles:      []string{"engineer"},
	}

	u30 = User{
		ID:         id3,
		VersionID:  id3,
		CreatedAt:  t3,
		UpdatedAt:  t3,
		GivenName:  "User_3",
		FamilyName: "Family_3",
		Email:      "test_user_three@test.com",
		Status:     ENABLED,
		Roles:      []string{"assistant", "analyst"},
		OrgID:      orgID3,
		OrgName:    "Test Organization 3",
	}

	u40 = User{
		ID:         id4,
		VersionID:  id4,
		CreatedAt:  t4,
		UpdatedAt:  t4,
		GivenName:  "User_4",
		FamilyName: "Family_4",
		Email:      "test_user_four@test.com",
		Status:     ENABLED,
		Roles:      []string{"assistant", "manager", "vp"},
		OrgID:      orgID4,
		OrgName:    "Test Organization 4",
	}

	u50 = User{
		ID:         id5,
		VersionID:  id5,
		CreatedAt:  t5,
		UpdatedAt:  t5,
		GivenName:  "User_5",
		FamilyName: "Family_5",
		Email:      "test_user_five@test.com",
		Status:     DISABLED,
		Roles:      []string{"assistant"},
		OrgID:      orgID4,
		OrgName:    "Test Organization 4",
	}

	u11 = User{
		ID:         id1,
		VersionID:  id1,
		CreatedAt:  t1,
		UpdatedAt:  t1,
		GivenName:  "User_1_updated",
		FamilyName: "Family_1_updated",
		Email:      "user_one_updated@test.com",
		Status:     ENABLED,
		Roles:      []string{"admin", "assistant", "manager", "owner"},
		OrgID:      orgID1,
		OrgName:    "Test Organization 1",
	}

	knownUsers      = []User{u11, u20, u30, u40, u50}
	knownIDs        = []string{id1, id2, id3, id4, id5}
	knownUserEmails = []string{u11.Email, u20.Email, u30.Email, u40.Email, u50.Email}
	knownOrgIDs     = []string{orgID1, orgID3, orgID4}
	knownOrgNames   = []string{"Test Organization 1", "Test Organization 3", "Test Organization 4"}
	allRoles        = []string{"admin", "analyst", "assistant", "engineer", "manager", "owner", "vp"}
	allStatuses     = []string{"DISABLED", "ENABLED", "PENDING"}
)

func TestMain(m *testing.M) {
	// Check table/row definitions
	if !service.Table.IsValid() {
		log.Fatal("invalid table configuration")
	}
	// Write known users
	for _, e := range []User{u10, u20, u30, u40, u50} {
		if _, err := service.Write(ctx, e); err != nil {
			log.Fatal(err)
		}
	}
	// Update first user
	var err error
	if u11, _, err = service.Update(ctx, u11); err != nil {
		log.Fatal(err)
	}
	// Run tests
	m.Run()
}

func TestCreateReadUpdateDelete(t *testing.T) {
	expect := assert.New(t)
	// Create user
	u, problems, err := service.Create(ctx, User{
		GivenName: "crud_test_user",
		Email:     "crud_user_email@test.com",
		Status:    PENDING,
	})
	expect.Empty(problems)

	if expect.NoError(err) {

		// User exists in organization table
		uExist := service.Exists(ctx, u.ID)
		expect.True(uExist)

		// Read the user
		uCheck, err := service.Read(ctx, u.ID)
		if expect.NoError(err) {
			// Check the organization
			expect.Equal(u, uCheck)
		}
		// Read the user as JSON
		uCheckJSON, err := service.ReadAsJSON(ctx, u.ID)
		if expect.NoError(err) {
			expect.Contains(string(uCheckJSON), u.ID)
		}
		// Update the user
		u.GivenName = "updated_crud_test_user"
		u.Email = "updated_crud_user_email@test.com"
		u.Status = ENABLED
		uUpdated, _, err := service.Update(ctx, u)
		if expect.NoError(err) {
			// Verify that the version ID has changed
			expect.NotEqual(uUpdated.ID, uUpdated.VersionID)
			expect.NotEqual(uUpdated.Status, PENDING)
		}
		// Delete version
		vDeleted, err := service.DeleteVersion(ctx, u.ID, u.VersionID)
		if expect.NoError(err) {
			expect.Equal(u.ID, vDeleted.ID)

			vExists := service.VersionExists(ctx, u.ID, u.VersionID)
			expect.False(vExists)
		}

		// Delete the user
		uDelete, err := service.Delete(ctx, u.ID)
		if expect.NoError(err) {
			// Check the user
			expect.Equal(uUpdated, uDelete)
		}
		// User does not exist
		uExist = service.Exists(ctx, u.ID)
		expect.False(uExist)

		// Read the user
		_, err = service.Read(ctx, u.ID)
		expect.ErrorIs(err, v.ErrNotFound, "expected ErrNotFound")
	}
}

func TestReadAsJSON(t *testing.T) {
	expect := assert.New(t)
	uCheckJSON, err := service.ReadAsJSON(ctx, id5)
	if expect.NoError(err) {
		expect.Contains(string(uCheckJSON), id5)
	}
}

func TestVersionExists(t *testing.T) {
	expect := assert.New(t)
	vExists := service.VersionExists(ctx, id4, id4)
	expect.True(vExists)
}

func TestReadVersion(t *testing.T) {
	expect := assert.New(t)
	vExist, err := service.ReadVersion(ctx, id1, id1)
	if expect.NoError(err) {
		expect.Equal(u10, vExist)
	}
}

func TestReadVersionAsJSON(t *testing.T) {
	expect := assert.New(t)
	vJSON, err := service.ReadVersionAsJSON(ctx, id3, id3)
	if expect.NoError(err) {
		expect.Contains(string(vJSON), id3)
	}
}

func TestReadVersions(t *testing.T) {
	expect := assert.New(t)
	vExist, err := service.ReadVersions(ctx, id1, false, 2, "")
	if expect.NoError(err) && expect.NotEmpty(vExist) {
		expect.Equal(u10, vExist[0])
	}
}

func TestReadVersionsAsJSON(t *testing.T) {
	expect := assert.New(t)
	vJSON, err := service.ReadVersionAsJSON(ctx, id2, id2)
	if expect.NoError(err) {
		expect.Contains(string(vJSON), id2)
	}
}

func TestReadAllVersions(t *testing.T) {
	expect := assert.New(t)
	allVersions, err := service.ReadAllVersions(ctx, id2)
	if expect.NoError(err) {
		expect.Equal(u20, allVersions[0])
	}
}

func TestReadAllVersionsAsJSON(t *testing.T) {
	expect := assert.New(t)
	versionsCheckJSON, err := service.ReadAllVersionsAsJSON(ctx, id3)
	if expect.NoError(err) {
		expect.Contains(string(versionsCheckJSON), id3)
	}
}

func TestReadIDs(t *testing.T) {
	expect := assert.New(t)
	ids, err := service.ReadIDs(ctx, false, 10, tuid.MinID)
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(ids), 3)
		expect.Subset(ids, knownIDs)
	}
}

func TestReadAllDs(t *testing.T) {
	expect := assert.New(t)
	allIDs, err := service.ReadAllIDs(ctx)
	if expect.NoError(err) {
		expect.Subset(allIDs, knownIDs)
	}
}

func TestReadNames(t *testing.T) {
	expect := assert.New(t)
	expectedNames := v.Map(knownUsers, func(u User) string { return u.String() })
	idsAndNames, err := service.ReadNames(ctx, false, 10, "-")

	if expect.NoError(err) && expect.NotEmpty(idsAndNames) {
		onlyNames := v.Map(idsAndNames, func(entry v.TextValue) string {
			return entry.Value
		})
		expect.Equal(len(onlyNames), len(knownIDs))
		expect.Subset(onlyNames, expectedNames)
	}
}

func TestReadAllNames(t *testing.T) {
	expect := assert.New(t)
	expectedName := []string{"User_3 Family_3 <test_user_three@test.com>"}
	idsAndNames, err := service.ReadAllNames(ctx, true)
	onlyNames := v.Map(idsAndNames, func(entry v.TextValue) string { return entry.Value })
	if expect.NoError(err) && expect.NotEmpty(onlyNames) {
		expect.Subset(onlyNames, expectedName)
	}
}

func TestFilterNames(t *testing.T) {
	expect := assert.New(t)
	expected := []v.TextValue{
		{
			Key:   u40.ID,
			Value: "User_4 Family_4 <test_user_four@test.com>",
		},
	}
	filteredNames, err := service.FilterNames(ctx, "User 4", true)
	if expect.NoError(err) && expect.NotEmpty(filteredNames) {
		expect.Subset(filteredNames, expected)
	}
}

func TestFilterNamesNegative(t *testing.T) {
	expect := assert.New(t)
	filteredNames, err := service.FilterNames(ctx, "no name", true)
	if expect.NoError(err) {
		expect.Empty(filteredNames)
	}
}

func TestReadUsers(t *testing.T) {
	expect := assert.New(t)
	users := service.ReadUsers(ctx, false, 5, "-")
	if expect.NotEmpty(users) {
		ids := v.Map(users, func(u User) string { return u.ID })
		expect.GreaterOrEqual(len(users), 5)
		expect.Subset(ids, knownIDs)
	}
}

func TestReadEmailAddresses(t *testing.T) {
	expect := assert.New(t)
	emails, err := service.ReadEmailAddresses(ctx, true, 10, "")
	if expect.NoError(err) && expect.NotEmpty(emails) {
		expect.Subset(emails, knownUserEmails)
	}
}

func TestReadAllEmailAddresses(t *testing.T) {
	expect := assert.New(t)
	emails, err := service.ReadAllEmailAddresses(ctx)
	if expect.NoError(err) && expect.NotEmpty(emails) {
		expect.Subset(emails, knownUserEmails)
	}
}

func TestReadUserByEmail(t *testing.T) {
	expect := assert.New(t)
	userByEmail, err := service.ReadUserByEmail(ctx, "user_one_updated@test.com")
	if expect.NoError(err) && expect.NotEmpty(userByEmail) {
		expect.Equal(userByEmail, u11)
	}
}

func TestReadAllUsersByEmail(t *testing.T) {
	expect := assert.New(t)
	allUsersByEmail, err := service.ReadAllUsersByEmail(ctx, "test_user_five@test.com")
	if expect.NoError(err) && expect.NotEmpty(allUsersByEmail) {
		expect.LessOrEqual(len(allUsersByEmail), 1)
		expect.Equal(allUsersByEmail[0], u50)
	}
}

func TestReadUserIDsByEmail(t *testing.T) {
	expect := assert.New(t)
	userIdsByEmail, err := service.ReadUserIDsByEmail(ctx, "test_user_five@test.com")
	if expect.NoError(err) && expect.NotEmpty(userIdsByEmail) {
		expect.LessOrEqual(len(userIdsByEmail), 1)
		expect.Equal(userIdsByEmail[0], u50.ID)
	}
}

func TestReadUserIDsByEmailNegative(t *testing.T) {
	expect := assert.New(t)
	userIdsByEmail, err := service.ReadUserIDsByEmail(ctx, "no_such_email@test.com")
	if expect.NoError(err) {
		expect.Empty(userIdsByEmail)
	}
}

//------------------------------------------------------------------------------
// Users by Organization
//------------------------------------------------------------------------------

func TestReadOrgs(t *testing.T) {
	expect := assert.New(t)
	orgIdAndName, err := service.ReadOrgs(ctx, false, 3, "")
	orgNames := v.Map(orgIdAndName, func(entry v.TextValue) string { return entry.Value })
	if expect.NoError(err) {
		expect.Equal(orgNames, knownOrgNames)
	}
}

func TestReadAllOrgs(t *testing.T) {
	expect := assert.New(t)
	orgIdAndName, err := service.ReadAllOrgs(ctx, true)
	if expect.NoError(err) {
		expect.Equal(len(orgIdAndName), 3)
	}
}

func TestReadOrgIDs(t *testing.T) {
	expect := assert.New(t)
	orgIds, err := service.ReadOrgIDs(ctx, false, 3, "")
	if expect.NoError(err) {
		expect.Equal(orgIds, knownOrgIDs)
	}
}

func TestReadAllOrgIDs(t *testing.T) {
	expect := assert.New(t)
	orgIds, err := service.ReadAllOrgIDs(ctx)
	if expect.NoError(err) {
		expect.Equal(orgIds, knownOrgIDs)
	}
}

func TestReadUsersByOrgID(t *testing.T) {
	expect := assert.New(t)
	users, err := service.ReadUsersByOrgID(ctx, orgID4, true, 2, "")
	if expect.NoError(err) && expect.NotEmpty(users) {
		expect.Subset(knownUsers, users)
	}
}

func TestReadUsersByOrgIDAsJSON(t *testing.T) {
	expect := assert.New(t)
	user, err := service.ReadUsersByOrgIDAsJSON(ctx, orgID3, true, 1, "")
	if expect.NoError(err) && expect.NotEmpty(user) {
		expect.Contains(string(user), u30.OrgID)
	}
}

func TestReadAllUsersByOrgID(t *testing.T) {
	expect := assert.New(t)
	users, err := service.ReadAllUsersByOrgID(ctx, orgID1)
	if expect.NoError(err) && expect.NotEmpty(users) {
		var numOfUsers []int
		var usersIds []string
		for i, v := range users {
			numOfUsers = append(numOfUsers, i)
			usersIds = append(usersIds, v.OrgID)
		}
		expect.GreaterOrEqual(len(numOfUsers), 1)
		expect.Subset(knownOrgIDs, usersIds)
	}
}

func TestReadAllUsersByOrgIDNegative(t *testing.T) {
	expect := assert.New(t)
	noUsers, err := service.ReadAllUsersByOrgID(ctx, orgID2)
	if expect.NoError(err) {
		expect.Empty(noUsers)
	}
}

func TestReadAllUsersByOrgIDAsJSON(t *testing.T) {
	expect := assert.New(t)
	users, err := service.ReadAllUsersByOrgIDAsJSON(ctx, orgID4)
	if expect.NoError(err) && expect.NotEmpty(users) {
		expect.Contains(string(users), u40.OrgID)
	}
}

//------------------------------------------------------------------------------
// Users by Role
//------------------------------------------------------------------------------

func TestReadRoles(t *testing.T) {
	expect := assert.New(t)
	roles, err := service.ReadRoles(ctx, false, 2, "o")
	if expect.NoError(err) {
		expect.Subset(allRoles, roles)
	}
}

func TestReadAllRoles(t *testing.T) {
	expect := assert.New(t)
	roles, err := service.ReadAllRoles(ctx)
	if expect.NoError(err) {
		expect.Equal(roles, allRoles)
	}
}

func TestReadUsersByRole(t *testing.T) {
	expect := assert.New(t)
	usersByRole, err := service.ReadUsersByRole(ctx, "vp", false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(usersByRole) {
		expect.Subset(knownUsers, usersByRole)
	}
}

func TestReadUsersByRoleAsJSON(t *testing.T) {
	expect := assert.New(t)
	usersByRole, err := service.ReadUsersByRoleAsJSON(ctx, "engineer", false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(usersByRole) {
		expect.Contains(string(usersByRole), u20.Roles[0])
	}
}

func TestReadAllUsersByRole(t *testing.T) {
	expect := assert.New(t)
	usersByRole, err := service.ReadAllUsersByRole(ctx, "assistant")
	if expect.NoError(err) && expect.NotEmpty(usersByRole) {
		var numOfUsers []int
		for i, _ := range usersByRole {
			numOfUsers = append(numOfUsers, i)
		}
		expect.GreaterOrEqual(len(numOfUsers), 4)
	}
}

func TestReadAllUsersByRoleNegative(t *testing.T) {
	expect := assert.New(t)
	noUsers, err := service.ReadAllUsersByOrgID(ctx, "ceo")
	if expect.NoError(err) {
		expect.Empty(noUsers)
	}
}

func TestReadAllUsersByRoleAsJSON(t *testing.T) {
	expect := assert.New(t)
	users, err := service.ReadAllUsersByRoleAsJSON(ctx, "admin")
	if expect.NoError(err) && expect.NotEmpty(users) {
		expect.Contains(string(users), u11.Roles[0])
	}
}

//------------------------------------------------------------------------------
// Users by Status
//------------------------------------------------------------------------------

func TestReadStatuses(t *testing.T) {
	expect := assert.New(t)
	// Read all statuses
	statuses, err := service.ReadStatuses(ctx, false, 3, "-")
	if expect.NoError(err) {
		expect.Equal(statuses, allStatuses)
	}
}

func TestReadAllStatuses(t *testing.T) {
	expect := assert.New(t)
	// Read all statuses
	statuses, err := service.ReadAllStatuses(ctx)
	if expect.NoError(err) {
		expect.Equal(statuses, allStatuses)
	}
}

func TestReadUsersByStatus(t *testing.T) {
	expect := assert.New(t)
	checkUsers, err := service.ReadUsersByStatus(ctx, "DISABLED", false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(checkUsers) {
		expect.Equal(u50, checkUsers[0])
	}
}

func TestReadUsersByStatusAsJSON(t *testing.T) {
	expect := assert.New(t)
	checkUsers, err := service.ReadUsersByStatusAsJSON(ctx, "DISABLED", false, 1, "")
	if expect.NoError(err) {
		expect.Contains(string(checkUsers), u50.ID)
	}
}

func TestReadAllUsersByStatus(t *testing.T) {
	expect := assert.New(t)
	checkUsers, err := service.ReadAllUsersByStatus(ctx, "ENABLED")
	if expect.NoError(err) && expect.NotEmpty(checkUsers) {
		var numOfUsers []int
		var usersIDs []string
		for i, v := range checkUsers {
			numOfUsers = append(numOfUsers, i)
			usersIDs = append(usersIDs, v.ID)
		}
		expect.GreaterOrEqual(len(numOfUsers), 4)
		expect.Subset(knownIDs, usersIDs)
	}

}

func TestReadAllUsersByStatusAsJSON(t *testing.T) {
	expect := assert.New(t)
	checkUsers, err := service.ReadAllUsersByStatusAsJSON(ctx, "DISABLED")
	if expect.NoError(err) {
		expect.Contains(string(checkUsers), u50.Status)
	}
}
