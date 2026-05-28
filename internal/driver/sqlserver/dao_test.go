//go:build integration

package sqlserver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcmssql "github.com/testcontainers/testcontainers-go/modules/mssql"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kopecmaciej/vi-sql/internal/config"
	"github.com/kopecmaciej/vi-sql/internal/database"
	"github.com/kopecmaciej/vi-sql/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mssqlPass = "SqlServer1!"
	testDB    = "tui_sample_db"
)

var testDao *Dao

func TestMain(m *testing.M) {
	ctx := context.Background()

	mssqlContainer, err := tcmssql.Run(ctx,
		"mcr.microsoft.com/mssql/server:2022-latest",
		tcmssql.WithAcceptEULA(),
		tcmssql.WithPassword(mssqlPass),
		testutil.SeedBindMount(),
		testcontainers.WithWaitStrategyAndDeadline(120*time.Second,
			wait.ForListeningPort("1433/tcp"),
			wait.ForLog("Starting up database"),
		),
	)
	if err != nil {
		panic("failed to start mssql container: " + err.Error())
	}
	defer mssqlContainer.Terminate(ctx) //nolint:errcheck

	masterConnStr, err := mssqlContainer.ConnectionString(ctx, "encrypt=disable")
	if err != nil {
		panic("failed to get master connection string: " + err.Error())
	}

	masterCfg := &config.SQLConfig{Driver: "sqlserver", DSN: masterConnStr}
	masterClient := NewClient(masterCfg)
	if err := masterClient.Connect(ctx); err != nil {
		panic("failed to connect to master: " + err.Error())
	}

	if _, err := masterClient.DB.ExecContext(ctx, "CREATE DATABASE "+testDB); err != nil {
		panic("failed to create test database: " + err.Error())
	}
	masterClient.Close()

	dbConnStr, err := mssqlContainer.ConnectionString(ctx, "database="+testDB+"&encrypt=disable")
	if err != nil {
		panic("failed to get db connection string: " + err.Error())
	}

	cfg := &config.SQLConfig{Driver: "sqlserver", DSN: dbConnStr, Timeout: 60}
	client := NewClient(cfg)
	if err := client.Connect(ctx); err != nil {
		panic("failed to connect to " + testDB + ": " + err.Error())
	}
	defer client.Close()

	if err := loadSampleSQL(ctx, client, testutil.SampleSQLServerPath()); err != nil {
		panic("failed to load sample SQL: " + err.Error())
	}

	testDao = NewDao(client)
	os.Exit(m.Run())
}

// loadSampleSQL reads the SQL Server sample file, splits it into batches on
// lines that are exactly "GO", and executes each non-empty batch.
func loadSampleSQL(ctx context.Context, client *Client, sqlPath string) error {
	content, err := os.ReadFile(sqlPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", sqlPath, err)
	}
	for i, batch := range splitGO(string(content)) {
		if _, err := client.DB.ExecContext(ctx, batch); err != nil {
			return fmt.Errorf("execute batch %d: %w\n--- batch ---\n%.200s", i, err, batch)
		}
	}
	return nil
}

// splitGO splits T-SQL source on lines that consist solely of "GO"
// (case-insensitive, trimmed). Empty batches are dropped.
func splitGO(sql string) []string {
	var batches []string
	var cur strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "go") {
			if b := strings.TrimSpace(cur.String()); b != "" {
				batches = append(batches, b)
			}
			cur.Reset()
		} else {
			cur.WriteString(line + "\n")
		}
	}
	if b := strings.TrimSpace(cur.String()); b != "" {
		batches = append(batches, b)
	}
	return batches
}

// --- Schema browsing ---

func TestListSchemas_MultipleSchemas(t *testing.T) {
	ctx := context.Background()

	schemas, err := testDao.ListSchemas(ctx, "")
	require.NoError(t, err)
	require.NotEmpty(t, schemas)

	names := make([]string, len(schemas))
	for i, s := range schemas {
		names[i] = s.Schema
	}
	assert.Contains(t, names, "auth")
	assert.Contains(t, names, "catalog")
}

func TestListSchemas_WithFilter(t *testing.T) {
	ctx := context.Background()

	schemas, err := testDao.ListSchemas(ctx, "auth")
	require.NoError(t, err)
	require.NotEmpty(t, schemas)
	for _, s := range schemas {
		assert.Contains(t, s.Schema, "auth")
	}
}

// --- Table structure ---

func TestGetTableColumns_WithSQLServerTypes(t *testing.T) {
	ctx := context.Background()

	cols, err := testDao.GetTableColumns(ctx, "auth", "users")
	require.NoError(t, err)
	require.NotEmpty(t, cols)
	for _, c := range cols {
		assert.NotEmpty(t, c.Name)
		assert.NotEmpty(t, c.DataType)
	}
}

func TestGetTableConstraints(t *testing.T) {
	ctx := context.Background()

	// auth.users has a PRIMARY KEY constraint.
	constraints, err := testDao.GetTableConstraints(ctx, "auth", "users")
	require.NoError(t, err)
	require.NotEmpty(t, constraints)
}

func TestGetTableForeignKeys(t *testing.T) {
	ctx := context.Background()

	// auth.user_roles has FKs to users and roles.
	fks, err := testDao.GetTableForeignKeys(ctx, "auth", "user_roles")
	require.NoError(t, err)
	require.NotEmpty(t, fks)
}

// --- Row CRUD ---

func TestListRows_BasicPagination(t *testing.T) {
	ctx := context.Background()

	state := database.NewTableState("auth", "users")
	state.BatchSize = 3

	_, rows, err := testDao.FetchTableRows(ctx, state, "", "")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(rows), 3)
}

func TestExecuteQuery_Select(t *testing.T) {
	ctx := context.Background()

	rows, cols, err := testDao.ExecuteQuery(ctx,
		"SELECT TOP 2 * FROM [auth].[users]")
	require.NoError(t, err)
	assert.NotEmpty(t, cols)
	_ = rows
}

func TestExecuteStatement_CreateDrop(t *testing.T) {
	ctx := context.Background()

	affected, err := testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.ms_stmt_test','U') IS NOT NULL DROP TABLE dbo.ms_stmt_test; "+
			"CREATE TABLE dbo.ms_stmt_test (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY)")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, affected, int64(0))

	_, err = testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.ms_stmt_test','U') IS NOT NULL DROP TABLE dbo.ms_stmt_test")
	require.NoError(t, err)
}

func TestExplainQuery_ReturnsOutput(t *testing.T) {
	ctx := context.Background()

	plan, err := testDao.ExplainPlan(ctx, "SELECT 1")
	require.NoError(t, err)
	assert.NotEmpty(t, plan)
}

// --- Indexes ---

func TestCreateIndex_GetIndexes_DropIndex(t *testing.T) {
	ctx := context.Background()

	_, err := testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.idx_ms_test','U') IS NOT NULL DROP TABLE dbo.idx_ms_test; "+
			"CREATE TABLE dbo.idx_ms_test (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY, name NVARCHAR(255))")
	require.NoError(t, err)
	defer testDao.ExecuteStatement(ctx, "IF OBJECT_ID('dbo.idx_ms_test','U') IS NOT NULL DROP TABLE dbo.idx_ms_test") //nolint:errcheck

	def := database.IndexDefinition{
		Name:     "idx_ms_test_name",
		Columns:  []string{"name"},
		IsUnique: false,
	}
	err = testDao.CreateIndex(ctx, "dbo", "idx_ms_test", def)
	require.NoError(t, err)

	indexes, err := testDao.GetIndexes(ctx, "dbo", "idx_ms_test")
	require.NoError(t, err)

	var found bool
	for _, idx := range indexes {
		if idx.Name == "idx_ms_test_name" {
			found = true
		}
	}
	assert.True(t, found, "created index should appear in GetIndexes")

	err = testDao.DropIndex(ctx, "dbo", "idx_ms_test_name")
	require.NoError(t, err)
}

// --- DDL ---

func TestCreateTable_And_DropTable(t *testing.T) {
	ctx := context.Background()

	ddl := "CREATE TABLE [dbo].[ms_ddl_test] ([id] INT NOT NULL IDENTITY(1,1) PRIMARY KEY, [label] NVARCHAR(255) NOT NULL)"
	err := testDao.CreateTable(ctx, "dbo", ddl)
	require.NoError(t, err)

	schemas, err := testDao.ListSchemas(ctx, "dbo")
	require.NoError(t, err)
	var found bool
	for _, s := range schemas {
		if s.Schema == "dbo" {
			for _, tbl := range s.Tables {
				if tbl == "ms_ddl_test" {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "created table should appear in schema listing")

	err = testDao.DropTable(ctx, "dbo", "ms_ddl_test")
	require.NoError(t, err)
}

func TestTruncateTable(t *testing.T) {
	ctx := context.Background()

	_, err := testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.trunc_ms_test','U') IS NOT NULL DROP TABLE dbo.trunc_ms_test; "+
			"CREATE TABLE dbo.trunc_ms_test (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY)")
	require.NoError(t, err)
	defer testDao.ExecuteStatement(ctx, "IF OBJECT_ID('dbo.trunc_ms_test','U') IS NOT NULL DROP TABLE dbo.trunc_ms_test") //nolint:errcheck

	_, err = testDao.ExecuteStatement(ctx,
		"INSERT INTO dbo.trunc_ms_test DEFAULT VALUES")
	require.NoError(t, err)

	err = testDao.TruncateTable(ctx, "dbo", "trunc_ms_test")
	require.NoError(t, err)

	state := database.NewTableState("dbo", "trunc_ms_test")
	state.BatchSize = 100
	_, rows, err := testDao.FetchTableRows(ctx, state, "", "")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

// --- Server info ---

func TestGetServerInfo_ReturnsVersion(t *testing.T) {
	ctx := context.Background()

	info, err := testDao.GetServerInfo(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Version, "SQL Server")
}

func TestGetActiveSessions_NonNegative(t *testing.T) {
	ctx := context.Background()

	sessions, err := testDao.GetActiveSessions(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, sessions, int64(0))
}

// --- Autocomplete ---

func TestGetTableColumnNames(t *testing.T) {
	ctx := context.Background()

	names, err := testDao.GetTableColumnNames(ctx, "auth", "users")
	require.NoError(t, err)
	assert.NotEmpty(t, names)
}

// --- Driver info ---

func TestCommonDataTypes_NonEmpty(t *testing.T) {
	types := testDao.CommonDataTypes()
	assert.NotEmpty(t, types)
}

func TestDefaultCreateTableDDL_ContainsTableName(t *testing.T) {
	ddl := testDao.DefaultCreateTableDDL("dbo", "my_ms_table")
	assert.Contains(t, ddl, "my_ms_table")
}

// --- GetTableColumns ---

func TestGetTableColumns_IdentityColumnIsAutoGenerated(t *testing.T) {
	ctx := context.Background()

	_, err := testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.identity_col_test','U') IS NOT NULL DROP TABLE dbo.identity_col_test; "+
			"CREATE TABLE dbo.identity_col_test (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY, name NVARCHAR(100) NOT NULL)")
	require.NoError(t, err)
	defer testDao.ExecuteStatement(ctx, "DROP TABLE dbo.identity_col_test") //nolint:errcheck

	cols, err := testDao.GetTableColumns(ctx, "dbo", "identity_col_test")
	require.NoError(t, err)

	var idCol, nameCol *database.ColumnInfo
	for i := range cols {
		switch cols[i].Name {
		case "id":
			idCol = &cols[i]
		case "name":
			nameCol = &cols[i]
		}
	}
	require.NotNil(t, idCol)
	require.NotNil(t, nameCol)
	assert.True(t, idCol.IsAutoGenerated, "IDENTITY column must be marked IsAutoGenerated")
	assert.False(t, nameCol.IsAutoGenerated, "plain column must not be marked IsAutoGenerated")
}

// --- Row CRUD (insert / update / delete) ---

func TestInsertRow_UpdateRow_DeleteRow(t *testing.T) {
	ctx := context.Background()

	_, err := testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.crud_ms_test','U') IS NOT NULL DROP TABLE dbo.crud_ms_test; "+
			"CREATE TABLE dbo.crud_ms_test (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY, label NVARCHAR(255) NOT NULL)")
	require.NoError(t, err)
	defer testDao.ExecuteStatement(ctx, "IF OBJECT_ID('dbo.crud_ms_test','U') IS NOT NULL DROP TABLE dbo.crud_ms_test") //nolint:errcheck

	pk, err := testDao.InsertRow(ctx, "dbo", "crud_ms_test", database.Row{"label": "hello"})
	require.NoError(t, err)

	rows, _, err := testDao.ExecuteQuery(ctx,
		fmt.Sprintf("SELECT label FROM dbo.crud_ms_test WHERE id = %v", pk.Columns["id"]))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	original := rows[0]
	assert.Equal(t, "hello", original["label"])

	updated := make(database.Row)
	for k, v := range original {
		updated[k] = v
	}
	updated["label"] = "world"
	updated["id"] = pk.Columns["id"]
	err = testDao.UpdateRow(ctx, "dbo", "crud_ms_test", pk, original, updated)
	require.NoError(t, err)

	rows, _, err = testDao.ExecuteQuery(ctx,
		fmt.Sprintf("SELECT label FROM dbo.crud_ms_test WHERE id = %v", pk.Columns["id"]))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "world", rows[0]["label"])

	err = testDao.DeleteRows(ctx, "dbo", "crud_ms_test", []database.PrimaryKey{pk})
	require.NoError(t, err)

	rows, _, err = testDao.ExecuteQuery(ctx,
		fmt.Sprintf("SELECT id FROM dbo.crud_ms_test WHERE id = %v", pk.Columns["id"]))
	require.NoError(t, err)
	assert.Empty(t, rows, "row should not exist after delete")
}

// --- FetchTableRows with filtering and ordering ---

func TestListRows_WithWhere(t *testing.T) {
	ctx := context.Background()

	_, err := testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.filter_ms_test','U') IS NOT NULL DROP TABLE dbo.filter_ms_test; "+
			"CREATE TABLE dbo.filter_ms_test (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY, status NVARCHAR(50) NOT NULL)")
	require.NoError(t, err)
	defer testDao.ExecuteStatement(ctx, "IF OBJECT_ID('dbo.filter_ms_test','U') IS NOT NULL DROP TABLE dbo.filter_ms_test") //nolint:errcheck

	_, err = testDao.ExecuteStatement(ctx,
		"INSERT INTO dbo.filter_ms_test (status) VALUES ('active'),('inactive'),('active')")
	require.NoError(t, err)

	state := database.NewTableState("dbo", "filter_ms_test")
	state.BatchSize = 100

	_, rows, err := testDao.FetchTableRows(ctx, state, "status = 'active'", "")
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	for _, r := range rows {
		assert.Equal(t, "active", r["status"])
	}
}

func TestListRows_WithOrderBy(t *testing.T) {
	ctx := context.Background()

	_, err := testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.order_ms_test','U') IS NOT NULL DROP TABLE dbo.order_ms_test; "+
			"CREATE TABLE dbo.order_ms_test (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY, label NVARCHAR(255) NOT NULL)")
	require.NoError(t, err)
	defer testDao.ExecuteStatement(ctx, "IF OBJECT_ID('dbo.order_ms_test','U') IS NOT NULL DROP TABLE dbo.order_ms_test") //nolint:errcheck

	_, err = testDao.ExecuteStatement(ctx,
		"INSERT INTO dbo.order_ms_test (label) VALUES ('charlie'),('alpha'),('bravo')")
	require.NoError(t, err)

	state := database.NewTableState("dbo", "order_ms_test")
	state.BatchSize = 100

	_, rows, err := testDao.FetchTableRows(ctx, state, "", "label ASC")
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	for i := 1; i < len(rows); i++ {
		prev := rows[i-1]["label"].(string)
		curr := rows[i]["label"].(string)
		assert.LessOrEqual(t, prev, curr, "rows should be ordered by label ASC")
	}
}

// --- RenameTable ---

func TestRenameTable_And_RenameBack(t *testing.T) {
	ctx := context.Background()

	_, err := testDao.ExecuteStatement(ctx,
		"IF OBJECT_ID('dbo.rename_ms_src','U') IS NOT NULL DROP TABLE dbo.rename_ms_src; "+
			"CREATE TABLE dbo.rename_ms_src (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY)")
	require.NoError(t, err)
	defer testDao.ExecuteStatement(ctx, "IF OBJECT_ID('dbo.rename_ms_src','U') IS NOT NULL DROP TABLE dbo.rename_ms_src") //nolint:errcheck
	defer testDao.ExecuteStatement(ctx, "IF OBJECT_ID('dbo.rename_ms_dst','U') IS NOT NULL DROP TABLE dbo.rename_ms_dst") //nolint:errcheck

	err = testDao.RenameTable(ctx, "dbo", "rename_ms_src", "rename_ms_dst")
	require.NoError(t, err)

	schemas, err := testDao.ListSchemas(ctx, "dbo")
	require.NoError(t, err)
	var tables []string
	for _, s := range schemas {
		if s.Schema == "dbo" {
			tables = s.Tables
		}
	}
	assert.Contains(t, tables, "rename_ms_dst")
	assert.NotContains(t, tables, "rename_ms_src")

	err = testDao.RenameTable(ctx, "dbo", "rename_ms_dst", "rename_ms_src")
	require.NoError(t, err)
}

func TestGetEstimatedRowCount(t *testing.T) {
	ctx := context.Background()

	count, _, err := testDao.GetEstimatedRowCount(ctx, "auth", "users")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
}

func TestRenameColumn(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, testDao.CreateTable(ctx, "dbo",
		"CREATE TABLE dbo.rename_col_ms (id INT NOT NULL IDENTITY(1,1) PRIMARY KEY, old_name NVARCHAR(255))"))
	defer testDao.DropTable(ctx, "dbo", "rename_col_ms") //nolint:errcheck

	err := testDao.RenameColumn(ctx, "dbo", "rename_col_ms", "old_name", "new_name")
	require.NoError(t, err)

	names, err := testDao.GetTableColumnNames(ctx, "dbo", "rename_col_ms")
	require.NoError(t, err)
	assert.Contains(t, names, "new_name")
	assert.NotContains(t, names, "old_name")
}

// --- FetchQueryRows ---

func TestListQueryRows_WithPagination(t *testing.T) {
	ctx := context.Background()

	_, rows, cols, err := testDao.FetchQueryRows(ctx,
		"SELECT * FROM [auth].[users]", 2, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, cols)
	assert.LessOrEqual(t, len(rows), 2)
}

func TestListQueryRows_NoLimit_Paginates(t *testing.T) {
	ctx := context.Background()

	const batch = 3
	_, rows, _, err := testDao.FetchQueryRows(ctx, "SELECT * FROM [auth].[users]", batch, 0)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(rows), batch, "first page must not exceed batch size")
	assert.NotEmpty(t, rows)

	_, rows2, _, err := testDao.FetchQueryRows(ctx, "SELECT * FROM [auth].[users]", batch, batch)
	require.NoError(t, err)
	assert.Len(t, rows2, batch, "second page should also have a full batch")
}

func TestListQueryRows_WithUserLimit_Paginates(t *testing.T) {
	ctx := context.Background()

	const batch = 2

	_, page1, _, err := testDao.FetchQueryRows(ctx,
		"SELECT TOP 5 * FROM [auth].[users]", batch, 0)
	require.NoError(t, err)
	assert.Len(t, page1, batch, "first page should return batch rows")

	_, page2, _, err := testDao.FetchQueryRows(ctx,
		"SELECT TOP 5 * FROM [auth].[users]", batch, batch)
	require.NoError(t, err)
	assert.Len(t, page2, batch, "second page should return batch rows")

	// Third page: offset=4 (batch*2), so only 1 row remains out of TOP 5.
	_, page3, _, err := testDao.FetchQueryRows(ctx,
		"SELECT TOP 5 * FROM [auth].[users]", batch, batch*2)
	require.NoError(t, err)
	assert.Len(t, page3, 1, "last page should have the remainder row")
}
