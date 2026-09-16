package database

import (
	"context"
	"errors"
)

var ErrReadOnly = errors.New("connection is read-only")

// ReadOnlyDriver wraps a Driver and rejects every mutating operation while
// delegating all reads to the embedded Driver.
type ReadOnlyDriver struct {
	Driver
}

func NewReadOnlyDriver(d Driver) Driver {
	return &ReadOnlyDriver{Driver: d}
}

func (r *ReadOnlyDriver) InsertRow(ctx context.Context, schema, table string, row Row) (PrimaryKey, error) {
	return PrimaryKey{}, ErrReadOnly
}

func (r *ReadOnlyDriver) UpdateRow(ctx context.Context, schema, table string, pk PrimaryKey, original, updated Row) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) UpdateRows(ctx context.Context, schema, table string, updates []RowUpdate) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) DeleteRows(ctx context.Context, schema, table string, pks []PrimaryKey) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) CreateTable(ctx context.Context, schema, ddl string) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) DropTable(ctx context.Context, schema, table string) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) RenameTable(ctx context.Context, schema, old, newName string) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) RenameColumn(ctx context.Context, schema, table, old, newName string) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) TruncateTable(ctx context.Context, schema, table string) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) CreateIndex(ctx context.Context, schema, table string, def IndexDefinition) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) DropIndex(ctx context.Context, schema, indexName string) error {
	return ErrReadOnly
}

func (r *ReadOnlyDriver) ExecuteStatement(ctx context.Context, stmt string) (int64, error) {
	return 0, ErrReadOnly
}
