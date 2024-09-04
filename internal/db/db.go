package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"reflect"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tursodatabase/libsql-client-go/libsql"
	"github.com/tursodatabase/libsql-client-go/sqliteparserutils"

	"github.com/libsql/libsql-shell-go/pkg/shell/enums"
	"github.com/libsql/libsql-shell-go/pkg/shell/shellerrors"
)

type driver int

const (
	libsqlDriver driver = iota
	sqlite3Driver
)

type Db struct {
	Uri       string
	AuthToken string

	sqlDb     *sql.DB
	driver    driver
	urlScheme string

	cancelRunningQuery func()
}

func (db *Db) IsRemote() bool {
	return db.driver == libsqlDriver
}

type StatementsResult struct {
	StatementResultCh chan StatementResult
}

type StatementResult struct {
	ColumnNames []string
	RowCh       chan rowResult
	Err         error
	Query       string
}

func newStatementResult(columnNames []string, rowCh chan rowResult, query string) *StatementResult {
	return &StatementResult{ColumnNames: columnNames, RowCh: rowCh, Query: query}
}

func newStatementResultWithError(err error) *StatementResult {
	treatedErr := treatDbError(err)
	return &StatementResult{Err: treatedErr}
}

type rowResult struct {
	Row []interface{}
	Err error
}

func newRowResult(row []interface{}) *rowResult {
	return &rowResult{Row: row}
}

func newRowResultWithError(err error) *rowResult {
	treatedErr := treatDbError(err)
	return &rowResult{Err: treatedErr}
}

func NewDb(dbUri, authToken, proxy string, schemaDb bool) (*Db, error) {
	var err error

	var db = Db{Uri: dbUri, AuthToken: authToken}

	if IsUrl(dbUri) {
		var validSqldUrl bool
		if validSqldUrl, db.urlScheme = IsValidSqldUrl(dbUri); validSqldUrl {
			db.driver = libsqlDriver
			var options []libsql.Option
			if authToken != "" {
				options = append(options, libsql.WithAuthToken(authToken))
			}
			if proxy != "" {
				options = append(options, libsql.WithProxy(proxy))
			}
			if schemaDb {
				options = append(options, libsql.WithSchemaDb(schemaDb))
			}
			connector, err := libsql.NewConnector(dbUri, options...)
			if err != nil {
				return nil, err
			}
			db.sqlDb = sql.OpenDB(connector)
		} else {
			return nil, &shellerrors.ProtocolError{}
		}
	} else {
		db.driver = sqlite3Driver
		db.sqlDb, err = sql.Open("sqlite3", dbUri)
	}
	if err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *Db) TestConnection() error {
	_, err := db.sqlDb.Exec("SELECT 1;")
	if err != nil {
		return fmt.Errorf("failed to connect to database. err: %v", err)
	}
	return nil
}

func (db *Db) Close() {
	db.sqlDb.Close()
}

func (db *Db) ExecuteStatements(statementsString string) (StatementsResult, error) {
	queries := db.prepareStatementsIntoQueries(statementsString)

	statementResultCh := make(chan StatementResult)

	go func() {
		defer close(statementResultCh)
		db.executeQueriesAndPopulateChannel(queries, statementResultCh)
	}()

	return StatementsResult{StatementResultCh: statementResultCh}, nil
}

func (db *Db) executeQueriesAndPopulateChannel(queries []string, statementResultCh chan StatementResult) {
	for _, query := range queries {
		if shouldContinue := db.executeQuery(query, statementResultCh); !shouldContinue {
			return
		}
	}
}

func (db *Db) ExecuteAndPrintStatements(statementsString string, outF io.Writer, withoutHeader bool, printMode enums.PrintMode) error {
	result, err := db.ExecuteStatements(statementsString)
	if err != nil {
		return err
	}

	err = PrintStatementsResult(result, outF, withoutHeader, printMode)
	if err != nil {
		return err
	}

	return nil
}

func (db *Db) executeQuery(query string, statementResultCh chan StatementResult) (queryEndedWithoutError bool) {
	if strings.TrimSpace(query) == "" {
		return true
	}

	ctx, cancel := context.WithCancel(context.Background())
	db.cancelRunningQuery = cancel

	rows, err := db.sqlDb.QueryContext(ctx, query)
	if err != nil {
		statementResultCh <- *newStatementResultWithError(err)

		return false
	}

	defer rows.Close()

	return readQueryResults(rows, statementResultCh, query)
}

func (db *Db) prepareStatementsIntoQueries(statementsString string) []string {
	// sqlite3 driver just run the first query that we send. So we must split the statements and send them one by one
	// e.g If we execute query "select 1; select 2;" with it, just the first one ("select 1;") would be executed
	//
	// libsql driver doesn't accept multiple statements if using websocket connection
	mustSplitStatementsIntoMultipleQueries :=
		db.driver == sqlite3Driver ||
			db.driver == libsqlDriver && (db.urlScheme == "libsql" || db.urlScheme == "wss" || db.urlScheme == "ws")

	if mustSplitStatementsIntoMultipleQueries {
		stmts, _ := sqliteparserutils.SplitStatement(statementsString)
		return stmts
	}

	return []string{statementsString}
}

func getColumnNames(rows *sql.Rows) ([]string, error) {
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	columnNames := make([]string, 0, len(columnTypes))
	for _, columnType := range columnTypes {
		columnNames = append(columnNames, columnType.Name())
	}

	return columnNames, nil
}

func getColumnTypes(rows *sql.Rows) ([]reflect.Type, error) {
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	types := make([]reflect.Type, len(columnTypes))

	for i, ct := range columnTypes {
		types[i] = ct.ScanType()
	}

	return types, nil
}

func readQueryResults(queryRows *sql.Rows, statementResultCh chan StatementResult, query string) (shouldContinue bool) {
	queries, _ := sqliteparserutils.SplitStatement(query)
	queryIndex := 0
	hasResultSetToRead := true
	for hasResultSetToRead {
		query := queries[queryIndex]
		if shouldContinue := readQueryResultSet(queryRows, statementResultCh, query); !shouldContinue {
			return false
		}

		hasResultSetToRead = queryRows.NextResultSet()
		queryIndex++
	}

	if err := queryRows.Err(); err != nil {
		statementResultCh <- *newStatementResultWithError(err)
		return false
	}

	return true
}

func readQueryResultSet(queryRows *sql.Rows, statementResultCh chan StatementResult, query string) (shouldContinue bool) {
	columnNames, err := getColumnNames(queryRows)
	if err != nil {
		statementResultCh <- *newStatementResultWithError(err)
		return false
	}

	columnTypes, err := getColumnTypes(queryRows)
	if err != nil {
		statementResultCh <- *newStatementResultWithError(err)
		return false
	}

	columnNamesLen := len(columnNames)
	columnPointers := make([]interface{}, columnNamesLen)
	for i, t := range columnTypes {
		if t.Kind() == reflect.Struct {
			columnPointers[i] = reflect.New(t).Interface()
		} else {
			columnPointers[i] = new(interface{})
		}
	}

	rowCh := make(chan rowResult)
	defer close(rowCh)

	statementResultCh <- *newStatementResult(columnNames, rowCh, query)

	for queryRows.Next() {
		err = queryRows.Scan(columnPointers...)
		if err != nil {
			rowCh <- *newRowResultWithError(err)
			return false
		}

		rowData := make([]interface{}, len(columnTypes))
		for i, ptr := range columnPointers {
			val := reflect.ValueOf(ptr).Elem()
			rowData[i] = val.Interface()
		}
		rowCh <- *newRowResult(rowData)
	}

	if err := queryRows.Err(); err != nil {
		rowCh <- *newRowResultWithError(err)
		return false
	}

	return true
}

func (db *Db) CancelQuery() {
	if db.cancelRunningQuery != nil {
		db.cancelRunningQuery()
	}
}

func treatDbError(originalErr error) error {
	err := originalErr

	if strings.Contains(err.Error(), "interactive transaction not allowed in HTTP queries") {
		err = &shellerrors.TransactionNotSupportedError{}
	}
	if strings.Contains(err.Error(), "context canceled") {
		err = &shellerrors.CancelQueryContextError{}
	}

	return err
}
