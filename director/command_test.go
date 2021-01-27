package director

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mdmdirector/mdmdirector/db"
	"github.com/mdmdirector/mdmdirector/types"
	"github.com/mdmdirector/mdmdirector/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestExampleHowToUseSqlmock(t *testing.T) {
	db, mock, err := sqlmock.New()
	defer db.Close()

	postgresMock, _, err := sqlmock.New()
	if err != nil {
		t.Errorf("Fail to get postgres mock")
	}

	_, err = gorm.Open(postgres.New(postgres.Config{Conn: postgresMock}), &gorm.Config{})
	assert.Equal(t, nil, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestClearCommands(t *testing.T) {
	// Old way of overriding flags... this doesn't work because flag.Parse() cannot be called multiple times
	// in the same process.
	// var tmp bool
	// os.Args = []string{"-clear-device-on-enroll", "true"}
	// flag.BoolVar(&tmp, "clear-device-on-enroll", true, "Deletes device profiles and install applications when a device enrolls")
	// flag.Parse()

	// New way of overriding flags:
	utils.FlagProvider = mockFlagBuilder{false}

	postgresMock, mockSpy, err := sqlmock.New()
	defer postgresMock.Close()
	if err != nil {
		t.Errorf("Fail to get postgres mock")
	}

	DB, err := gorm.Open(postgres.New(postgres.Config{Conn: postgresMock}), &gorm.Config{})
	db.DB = DB

	mockSpy.ExpectBegin()
	mockSpy.ExpectExec(`^DELETE FROM "commands" WHERE device_ud_id = \$1 AND NOT \(status = \$2 OR status = \$3\)`).WithArgs(
		"1234-5678-123456",
		"Error",
		"Acknowledged",
	).WillReturnResult(sqlmock.NewResult(0, 0))
	mockSpy.ExpectCommit()

	device := types.Device{
		SerialNumber: "C02ABCDEFGH",
		UDID:         "1234-5678-123456",
	}
	err = ClearCommands(&device)

	assert.Equal(t, nil, err)
}

func TestClearCommands_ClearDeviceOnEnroll(t *testing.T) {
	utils.FlagProvider = mockFlagBuilder{true}

	// Set up Database Mocks
	postgresMock, mockSpy, _ := sqlmock.New()
	defer postgresMock.Close()

	DB, err := gorm.Open(postgres.New(postgres.Config{Conn: postgresMock}), &gorm.Config{})
	db.DB = DB

	// Set up Database expectations
	mockSpy.ExpectBegin()
	mockSpy.ExpectExec(`^DELETE FROM "commands" WHERE device_ud_id = \$1 AND NOT \(status = \$2 OR status = \$3\)`).WithArgs(
		"1234-5678-123456",
		"Error",
		"Acknowledged",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mockSpy.ExpectCommit()

	mockSpy.ExpectBegin()
	mockSpy.ExpectExec(`^DELETE FROM "device_profiles" WHERE device_ud_id = \$1`).WithArgs(
		"1234-5678-123456",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mockSpy.ExpectCommit()

	mockSpy.ExpectBegin()
	mockSpy.ExpectExec(`^DELETE FROM "device_install_applications" WHERE device_ud_id = \$1`).WithArgs(
		"1234-5678-123456",
	).WillReturnResult(sqlmock.NewResult(0, 0))
	mockSpy.ExpectCommit()

	device := types.Device{
		SerialNumber: "C02ABCDEFGH",
		UDID:         "1234-5678-123456",
	}
	err = ClearCommands(&device)

	assert.Equal(t, nil, err)
}

func TestClearCommands_OnDeleteError(t *testing.T) {
	postgresMock, mockSpy, _ := sqlmock.New()
	defer postgresMock.Close()

	DB, err := gorm.Open(postgres.New(postgres.Config{Conn: postgresMock}), &gorm.Config{})
	db.DB = DB

	mockSpy.ExpectExec(`.*`).WithArgs(
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
	).WillReturnError(errors.New("database has gone away"))

	device := types.Device{
		SerialNumber: "C02ABCDEFGH",
		UDID:         "1234-5678-123456",
	}
	err = ClearCommands(&device)

	assert.NotEmpty(t, err)
	assert.Equal(t, "Failed to clear Command Queue for 1234-5678-123456: database has gone away", err.Error())
}

// // Test classes
type mockFlagBuilder struct {
	doClear bool
}

func (m mockFlagBuilder) ClearDeviceOnEnroll() bool {
	return m.doClear
}
