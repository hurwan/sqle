package model

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
)

func TestStorage_GetAuditPlans(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	InitMockStorage(mockDB)
	mock.ExpectQuery("SELECT * FROM `audit_plans` WHERE `audit_plans`.`deleted_at` IS NULL").
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("audit_plan_1"))
	mock.ExpectClose()
	aps, err := GetStorage().GetAuditPlans()
	assert.NoError(t, err)
	assert.Len(t, aps, 1)
	mockDB.Close()
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestStorage_GetAuditPlanByName(t *testing.T) {
	// 1. test record exist
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	InitMockStorage(mockDB)
	mock.ExpectQuery("SELECT * FROM `audit_plans` WHERE `audit_plans`.`deleted_at` IS NULL AND ((name = ?))").
		WithArgs("audit_plan_for_java_repo1").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("audit_plan_1"))
	mock.ExpectClose()
	ap, exist, err := GetStorage().GetAuditPlanByName("audit_plan_for_java_repo1")
	assert.NoError(t, err)
	assert.True(t, exist)
	assert.Equal(t, "audit_plan_1", ap.Name)
	mockDB.Close()
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// 2. test record not exist
	mockDB, mock, err = sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	InitMockStorage(mockDB)
	mock.ExpectQuery("SELECT * FROM `audit_plans` WHERE `audit_plans`.`deleted_at` IS NULL AND ((name = ?))").
		WithArgs("audit_plan_for_java_repo1").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectClose()
	_, exist, err = GetStorage().GetAuditPlanByName("audit_plan_for_java_repo1")
	assert.NoError(t, err)
	assert.False(t, exist)
	mockDB.Close()
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestStorage_GetAuditPlanSQLs(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	InitMockStorage(mockDB)

	mockAuditPlanRow := AuditPlan{Model: Model{ID: 1}, Name: "audit_plan_for_java_repo1"}

	mock.ExpectQuery("SELECT * FROM `audit_plans` WHERE `audit_plans`.`deleted_at` IS NULL AND ((name = ?))").
		WithArgs("audit_plan_for_java_repo1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(mockAuditPlanRow.ID, mockAuditPlanRow.Name))
	mock.ExpectQuery("SELECT * FROM `audit_plan_sqls_v2`  WHERE `audit_plan_sqls_v2`.`deleted_at` IS NULL AND ((audit_plan_id = ?))").
		WithArgs(mockAuditPlanRow.ID).
		WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}).AddRow("select * from t1 where id = ?").AddRow("select * from t2 where id = ?"))
	mock.ExpectClose()
	sqls, err := GetStorage().GetAuditPlanSQLs(mockAuditPlanRow.Name)
	assert.NoError(t, err)
	assert.Len(t, sqls, 2)
	mockDB.Close()
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)

	// 2. test update audit plan not exist
	mockDB, mock, err = sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	InitMockStorage(mockDB)
	mock.ExpectQuery("SELECT * FROM `audit_plans` WHERE `audit_plans`.`deleted_at` IS NULL AND ((name = ?))").
		WithArgs("audit_plan_for_java_repo1").
		WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectClose()
	_, err = GetStorage().GetAuditPlanSQLs("audit_plan_for_java_repo1")
	assert.Equal(t, gorm.ErrRecordNotFound, err)
	mockDB.Close()
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestStorage_OverrideAuditPlanSQLs(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	InitMockStorage(mockDB)

	ap := &AuditPlan{
		Model: Model{
			ID: 1,
		},
		Name: "test_ap_name",
	}

	sqls := []*AuditPlanSQLV2{
		{
			Fingerprint: "select * from t1 where id = ?",
			SQLContent:  "select * from t1 where id = 1",
			Info:        []byte(`{"counter": 1, "last_receive_timestamp": "mock time"}`),
		},
	}

	mock.ExpectQuery("SELECT * FROM `audit_plans` WHERE `audit_plans`.`deleted_at` IS NULL AND ((name = ?))").
		WithArgs(ap.Name).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(ap.ID, ap.Name))

	mock.ExpectBegin()
	// expect hard delete
	mock.ExpectExec("DELETE FROM `audit_plan_sqls_v2` WHERE (audit_plan_id = ?)").
		WithArgs(ap.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectExec("INSERT INTO `audit_plan_sqls_v2` (`audit_plan_id`,`fingerprint_md5`, `fingerprint`, `sql_content`, `info`) VALUES (?, ?, ?, ?, ?);").
		WithArgs(ap.ID, sqls[0].GetFingerprintMD5(), sqls[0].Fingerprint, sqls[0].SQLContent, sqls[0].Info).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = GetStorage().OverrideAuditPlanSQLs(ap.Name, sqls)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
