package database

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mattn/go-sqlite3"
)

// column mirrors one row of SQLite's PRAGMA table_info.
type column struct {
	name    string
	typ     string
	notNull bool
	dflt    sql.NullString
	pk      bool
}

func dflt(v string) sql.NullString { return sql.NullString{String: v, Valid: true} }

func TestInit_CreatesTablesMatchingSpec(t *testing.T) {
	db := newTestDB(t)

	tests := []struct {
		table string
		want  []column
	}{
		{
			table: "loan_customers",
			want: []column{
				{name: "customer_id", typ: "TEXT", pk: true},
				{name: "id_card_number", typ: "TEXT", notNull: true},
				{name: "full_name", typ: "TEXT", notNull: true},
				{name: "birth_date", typ: "TEXT", notNull: true},
				{name: "phone_number", typ: "TEXT", notNull: true},
				{name: "email", typ: "TEXT"},
				{name: "monthly_income", typ: "INTEGER", notNull: true, dflt: dflt("0")},
				{name: "address_street", typ: "TEXT", notNull: true},
				{name: "address_city", typ: "TEXT", notNull: true},
			},
		},
		{
			table: "loan_submissions",
			want: []column{
				{name: "submission_id", typ: "TEXT", pk: true},
				{name: "vehicle_type", typ: "TEXT", notNull: true},
				{name: "vehicle_brand", typ: "TEXT", notNull: true},
				{name: "vehicle_model", typ: "TEXT", notNull: true},
				{name: "vehicle_license_number", typ: "TEXT", notNull: true},
				{name: "vehicle_odometer", typ: "INTEGER", notNull: true, dflt: dflt("0")},
				{name: "manufacturing_year", typ: "INTEGER", notNull: true},
				{name: "proposed_loan_amount", typ: "INTEGER", notNull: true, dflt: dflt("0")},
				{name: "proposed_loan_tenure_month", typ: "INTEGER", notNull: true, dflt: dflt("0")},
				{name: "loan_status", typ: "TEXT", notNull: true},
				{name: "is_commercial_vehicle", typ: "BOOLEAN", notNull: true, dflt: dflt("FALSE")},
				{name: "created_at", typ: "INTEGER", notNull: true},
				{name: "updated_at", typ: "INTEGER", notNull: true},
				{name: "customer_id", typ: "TEXT", notNull: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			got := tableColumns(t, db, tt.table)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("columns of %s:\n got  %+v\n want %+v", tt.table, got, tt.want)
			}
		})
	}
}

func TestInit_RejectsDuplicateIDCardNumber(t *testing.T) {
	db := newTestDB(t)
	mustInsertCustomer(t, db, "cust-1", "ID-123")

	err := insertCustomer(db, "cust-2", "ID-123")

	assertConstraint(t, err, sqlite3.ErrConstraintUnique)
}

func TestInit_RejectsSubmissionForUnknownCustomer(t *testing.T) {
	db := newTestDB(t)

	err := insertSubmission(db, "sub-1", "no-such-customer")

	assertConstraint(t, err, sqlite3.ErrConstraintForeignKey)
}

func TestInit_DeletingCustomerDeletesOnlyTheirSubmissions(t *testing.T) {
	db := newTestDB(t)
	mustInsertCustomer(t, db, "cust-1", "ID-1")
	mustInsertCustomer(t, db, "cust-2", "ID-2")
	mustInsertSubmission(t, db, "sub-1", "cust-1")
	mustInsertSubmission(t, db, "sub-2", "cust-1")
	mustInsertSubmission(t, db, "sub-3", "cust-2")

	if _, err := db.Exec(`DELETE FROM loan_customers WHERE customer_id = ?`, "cust-1"); err != nil {
		t.Fatalf("delete customer: %v", err)
	}

	if n := countSubmissions(t, db, "cust-1"); n != 0 {
		t.Errorf("submissions left for deleted customer = %d, want 0", n)
	}
	if n := countSubmissions(t, db, "cust-2"); n != 1 {
		t.Errorf("submissions left for other customer = %d, want 1", n)
	}
}

func TestInit_RunTwiceKeepsSchemaAndData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alphaloan.db")
	first, err := Init(path)
	if err != nil {
		t.Fatalf("first Init: %v", err)
	}
	mustInsertCustomer(t, first, "cust-1", "ID-1")
	first.Close()

	second, err := Init(path)
	if err != nil {
		t.Fatalf("second Init: %v", err)
	}
	t.Cleanup(func() { second.Close() })

	var version int
	var dirty bool
	if err := second.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty); err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != 2 || dirty {
		t.Errorf("migration state = (version %d, dirty %t), want (2, false)", version, dirty)
	}
	var customers int
	if err := second.QueryRow(`SELECT COUNT(*) FROM loan_customers`).Scan(&customers); err != nil {
		t.Fatalf("count customers: %v", err)
	}
	if customers != 1 {
		t.Errorf("customers after restart = %d, want 1", customers)
	}
}

// The sqlite3 migration driver closes the *sql.DB it is given when the
// migrator is closed. Init must hand back a handle that is still usable.
func TestInit_ReturnsOpenHandle(t *testing.T) {
	db := newTestDB(t)

	if err := db.Ping(); err != nil {
		t.Errorf("Ping after Init: %v", err)
	}
}

func TestInit_FailsWhenDirectoryIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "alphaloan.db")

	db, err := Init(path)

	if err == nil {
		db.Close()
		t.Fatal("Init succeeded, want error for a path in a missing directory")
	}
}

func TestInit_FailsWhenFileIsNotADatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alphaloan.db")
	if err := os.WriteFile(path, []byte(strings.Repeat("not a database ", 64)), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	db, err := Init(path)

	if err == nil {
		db.Close()
		t.Fatal("Init succeeded, want error for a file that is not a SQLite database")
	}
}

func TestMigrations_DownThenUpRestoresSchema(t *testing.T) {
	db := newTestDB(t)
	m, err := newMigrator(db)
	if err != nil {
		t.Fatalf("newMigrator: %v", err)
	}

	if err := m.Down(); err != nil {
		t.Fatalf("Down: %v", err)
	}
	if n := countUserTables(t, db); n != 0 {
		t.Fatalf("tables after Down = %d, want 0", n)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("Up after Down: %v", err)
	}
	if n := countUserTables(t, db); n != 2 {
		t.Errorf("tables after Up = %d, want 2", n)
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Init(filepath.Join(t.TempDir(), "alphaloan.db"))
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func tableColumns(t *testing.T, db *sql.DB, table string) []column {
	t.Helper()
	rows, err := db.Query(`SELECT name, type, "notnull", dflt_value, pk FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatalf("table_info(%s): %v", table, err)
	}
	defer rows.Close()

	var cols []column
	for rows.Next() {
		var c column
		var pk int
		if err := rows.Scan(&c.name, &c.typ, &c.notNull, &c.dflt, &pk); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		c.pk = pk > 0
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info(%s): %v", table, err)
	}
	return cols
}

func insertCustomer(db *sql.DB, customerID, idCardNumber string) error {
	_, err := db.Exec(`
		INSERT INTO loan_customers
			(customer_id, id_card_number, full_name, birth_date, phone_number, address_street, address_city)
		VALUES (?, ?, 'Test Customer', '1990-01-01', '081200000000', 'Jl. Sudirman 1', 'Jakarta')`,
		customerID, idCardNumber)
	return err
}

func mustInsertCustomer(t *testing.T, db *sql.DB, customerID, idCardNumber string) {
	t.Helper()
	if err := insertCustomer(db, customerID, idCardNumber); err != nil {
		t.Fatalf("insert customer %s: %v", customerID, err)
	}
}

func insertSubmission(db *sql.DB, submissionID, customerID string) error {
	_, err := db.Exec(`
		INSERT INTO loan_submissions
			(submission_id, vehicle_type, vehicle_brand, vehicle_model, vehicle_license_number,
			 manufacturing_year, loan_status, created_at, updated_at, customer_id)
		VALUES (?, 'car', 'Toyota', 'Avanza', 'B 1234 XYZ', 2020, 'SUBMITTED', 0, 0, ?)`,
		submissionID, customerID)
	return err
}

func mustInsertSubmission(t *testing.T, db *sql.DB, submissionID, customerID string) {
	t.Helper()
	if err := insertSubmission(db, submissionID, customerID); err != nil {
		t.Fatalf("insert submission %s: %v", submissionID, err)
	}
}

func countSubmissions(t *testing.T, db *sql.DB, customerID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM loan_submissions WHERE customer_id = ?`, customerID).Scan(&n); err != nil {
		t.Fatalf("count submissions for %s: %v", customerID, err)
	}
	return n
}

func countUserTables(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name IN ('loan_customers', 'loan_submissions')`).Scan(&n)
	if err != nil {
		t.Fatalf("count tables: %v", err)
	}
	return n
}

func assertConstraint(t *testing.T, err error, want sqlite3.ErrNoExtended) {
	t.Helper()
	var sqliteErr sqlite3.Error
	if !errors.As(err, &sqliteErr) {
		t.Fatalf("err = %v, want sqlite3 constraint error %v", err, want)
	}
	if sqliteErr.ExtendedCode != want {
		t.Errorf("constraint = %v, want %v", sqliteErr.ExtendedCode, want)
	}
}
