//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

import (
	"github.com/go-jet/jet/v2/postgres"
)

var UserPrivateKeys = newUserPrivateKeysTable("public", "user_private_keys", "")

type userPrivateKeysTable struct {
	postgres.Table

	// Columns
	UserID       postgres.ColumnString
	PrivateKeyID postgres.ColumnString

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
}

type UserPrivateKeysTable struct {
	userPrivateKeysTable

	EXCLUDED userPrivateKeysTable
}

// AS creates new UserPrivateKeysTable with assigned alias
func (a UserPrivateKeysTable) AS(alias string) *UserPrivateKeysTable {
	return newUserPrivateKeysTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new UserPrivateKeysTable with assigned schema name
func (a UserPrivateKeysTable) FromSchema(schemaName string) *UserPrivateKeysTable {
	return newUserPrivateKeysTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new UserPrivateKeysTable with assigned table prefix
func (a UserPrivateKeysTable) WithPrefix(prefix string) *UserPrivateKeysTable {
	return newUserPrivateKeysTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new UserPrivateKeysTable with assigned table suffix
func (a UserPrivateKeysTable) WithSuffix(suffix string) *UserPrivateKeysTable {
	return newUserPrivateKeysTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newUserPrivateKeysTable(schemaName, tableName, alias string) *UserPrivateKeysTable {
	return &UserPrivateKeysTable{
		userPrivateKeysTable: newUserPrivateKeysTableImpl(schemaName, tableName, alias),
		EXCLUDED:             newUserPrivateKeysTableImpl("", "excluded", ""),
	}
}

func newUserPrivateKeysTableImpl(schemaName, tableName, alias string) userPrivateKeysTable {
	var (
		UserIDColumn       = postgres.StringColumn("user_id")
		PrivateKeyIDColumn = postgres.StringColumn("private_key_id")
		allColumns         = postgres.ColumnList{UserIDColumn, PrivateKeyIDColumn}
		mutableColumns     = postgres.ColumnList{UserIDColumn, PrivateKeyIDColumn}
	)

	return userPrivateKeysTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		UserID:       UserIDColumn,
		PrivateKeyID: PrivateKeyIDColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
