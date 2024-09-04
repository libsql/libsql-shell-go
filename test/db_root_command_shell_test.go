package main_test

import (
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/stretchr/testify/suite"

	"github.com/libsql/libsql-shell-go/test/utils"
)

type DBRootCommandShellSuite struct {
	suite.Suite

	dbUri     string
	authToken string
	tc        *utils.DbTestContext
}

func NewDBRootCommandShellSuite(dbUri string) *DBRootCommandShellSuite {
	return &DBRootCommandShellSuite{dbUri: dbUri}
}

func (s *DBRootCommandShellSuite) SetupSuite() {
	s.tc = utils.NewTestContext(s.T(), s.dbUri, s.authToken)
	s.tc.DropAllTables()
}

func (s *DBRootCommandShellSuite) TearDownTest() {
	s.tc.DropAllTables()
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithTwoTables_WhenCallDotTablesCommand_ExpectAListContainingTheTableNames() {
	s.tc.CreateEmptySimpleTable("simple_table")
	s.tc.CreateEmptySimpleTable("another_simple_table")

	outTables, errS, err := s.tc.ExecuteShell([]string{".tables"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outTables, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{"another_simple_table\nsimple_table"}}))
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithTwoTables_WhenCallDotSchemaCommand_ExpectAListContainingTheSchemas() {
	s.tc.CreateEmptySimpleTable("simple_table")
	s.tc.CreateEmptySimpleTable("another_simple_table")

	outSchema, errS, err := s.tc.ExecuteShell([]string{".schema"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outSchema, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{"CREATE TABLE another_simple_table (id INTEGER PRIMARY KEY, textField TEXT, intField INTEGER);\nCREATE TABLE simple_table (id INTEGER PRIMARY KEY, textField TEXT, intField INTEGER);"}}))
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithTwoTables_WhenCallDotSchemaCommandWithPattern_ExpectToReturnOneSchema() {
	s.tc.CreateEmptySimpleTable("simple_table")
	s.tc.CreateEmptySimpleTable("another_simple_table")

	outSchema, errS, err := s.tc.ExecuteShell([]string{".schema simple_table"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outSchema, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{"CREATE TABLE simple_table (id INTEGER PRIMARY KEY, textField TEXT, intField INTEGER);"}}))
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithThreeTables_WhenCallDotSchemaCommandWithPartialDbName_ExpectToReturnTwoSchemas() {
	s.tc.CreateEmptySimpleTable("simple_table")
	s.tc.CreateEmptySimpleTable("test_table_one")
	s.tc.CreateEmptySimpleTable("test_table_two")

	outSchema, errS, err := s.tc.ExecuteShell([]string{".schema test%"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outSchema, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{"CREATE TABLE test_table_one (id INTEGER PRIMARY KEY, textField TEXT, intField INTEGER);\nCREATE TABLE test_table_two (id INTEGER PRIMARY KEY, textField TEXT, intField INTEGER);"}}))
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithTwoTables_WhenCallDotSchemaCommandWithPatternThatDoesNotMatch_ExpectEmptyReturn() {
	s.tc.CreateEmptySimpleTable("simple_table")
	s.tc.CreateEmptySimpleTable("another_simple_table")

	outSchema, errS, err := s.tc.ExecuteShell([]string{".schema non_existing_table"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outSchema, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{""}}))
}

func (s *DBRootCommandShellSuite) Test_WhenCallDotHelpCommand_ExpectAListWithAllAvailableCommands() {
	outS, errS, err := s.tc.ExecuteShell([]string{".help"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	expectedHelp :=
		`.dump       Render database content as SQL
  .help       List of all available commands.
  .indexes    List indexes in a table or database
  .mode       Set output mode
  .quit       Exit this program
  .read       Execute commands from a file
  .schema     Show table schemas.
  .tables     List all existing tables in the database.`
	s.tc.Assert(outS, qt.Equals, expectedHelp)
}

func (s *DBRootCommandShellSuite) Test_GivenAEmptyDb_WhenCallDotReadCommand_ExpectToSeeATableWithOneEntry() {
	content := `CREATE TABLE IF NOT EXISTS testread (name TEXT);
		/* Comment in the middle of the file.*/
		INSERT INTO testread VALUES("test");
		
		SELECT * FROM testread;`
	file, filePath := s.tc.CreateTempFile(content)

	defer file.Close()

	outS, errS, err := s.tc.ExecuteShell([]string{".read " + filePath})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, utils.GetPrintTableOutput([]string{"NAME"}, [][]string{{"test"}}))
}

func (s *DBRootCommandShellSuite) Test_GivenAEmptyDb_WhenCallDotReadCommandPassingANonExistingFile_ExpectToReturnAnErrorMessage() {
	outS, errS, err := s.tc.ExecuteShell([]string{".read nonExistingFile.txt"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "Error: open nonExistingFile.txt: no such file or directory")
	s.tc.Assert(outS, qt.Equals, "")
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithTwoTables_WhenCreateTwoIndexesAndCallDotIndexesCommand_ExpectToReturnTheIndexes() {
	s.tc.CreateSimpleTable("simple_table", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})
	s.tc.CreateSimpleTable("simple_table_2", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})

	_, errS, err := s.tc.Execute("CREATE INDEX idx_textfield on simple_table (TextField);CREATE INDEX idx_intfield on simple_table_2 (IntField)")
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	outS, errS, err := s.tc.ExecuteShell([]string{".indexes"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{"idx_textfield\nidx_intfield"}}))
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithThreeTables_WhenCreateThreeIndexesAndCallDotIndexesCommandPassingExactTableName_ExpectToReturnJustOneIndex() {
	s.tc.CreateSimpleTable("simple_table", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})
	s.tc.CreateSimpleTable("simple_table_2", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})
	s.tc.CreateSimpleTable("simple_table_3", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})

	_, errS, err := s.tc.Execute("CREATE INDEX idx_textfield on simple_table (TextField);CREATE INDEX idx_intfield on simple_table_2 (IntField);CREATE INDEX idx_intfield_third_table on simple_table_3 (IntField)")
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	outS, errS, err := s.tc.ExecuteShell([]string{".indexes simple_table_3"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{"idx_intfield_third_table"}}))
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithThreeTables_WhenCreateThreeIndexesAndCallDotIndexesCommandPassingPartOfTableName_ExpectToReturnTwoIndexes() {
	s.tc.CreateSimpleTable("simple_table", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})
	s.tc.CreateSimpleTable("simple_table_2", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})
	s.tc.CreateSimpleTable("another_simple_table", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})

	_, errS, err := s.tc.Execute("CREATE INDEX idx_textfield on simple_table (TextField);CREATE INDEX idx_intfield on simple_table_2 (IntField);CREATE INDEX idx_intfield_another_table on another_simple_table (IntField)")
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	outS, errS, err := s.tc.ExecuteShell([]string{".indexes simple%"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{"idx_textfield\nidx_intfield"}}))
}

func (s *DBRootCommandShellSuite) Test_GivenADBWithATable_WhenCreateAIndexAndCallDotIndexesCommandPassingAWrongTableName_ExpectEmptyReturn() {
	s.tc.CreateSimpleTable("simple_table", []utils.SimpleTableEntry{{TextField: "value1", IntField: 1}, {TextField: "value2", IntField: 2}})

	_, errS, err := s.tc.Execute("CREATE INDEX idx_textfield on simple_table (TextField);")
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	outS, errS, err := s.tc.ExecuteShell([]string{".indexes nonExistingTable"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, utils.GetPrintTableOutput([]string{""}, [][]string{{""}}))
}

func (s *DBRootCommandShellSuite) Test_GivenAEmptyTable_WhenCallDotDumpCommand_ExpectNoErrors() {
	s.tc.CreateEmptyAllTypesTable("alltypes")

	outS, errS, err := s.tc.ExecuteShell([]string{".dump"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")
	prefix := "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\nCREATE TABLE"
	if !strings.HasSuffix(s.dbUri, "test.sqlite") {
		prefix += " if not exists"
	}
	expected := " alltypes (textNullable text, textNotNullable text NOT NULL, textWithDefault text DEFAULT 'defaultValue', \n\tintNullable INTEGER, intNotNullable INTEGER NOT NULL, intWithDefault INTEGER DEFAULT '0', \n\tfloatNullable REAL, floatNotNullable REAL NOT NULL, floatWithDefault REAL DEFAULT '0.0', \n\tunknownNullable NUMERIC, unknownNotNullable NUMERIC NOT NULL, unknownWithDefault NUMERIC DEFAULT 0.0, \n\tblobNullable BLOB, blobNotNullable BLOB NOT NULL, blobWithDefault BLOB DEFAULT 'x\"0\"');\nCOMMIT;"
	s.tc.AssertSqlEquals(outS, prefix+expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableConainingRandomFields_WhenInsertAndCallDotDumpCommand_ExpectNoErrors() {
	_, errS, err := s.tc.Execute(`CREATE TABLE alltypes (t text, i integer, r real, b blob);
	INSERT INTO alltypes VALUES ('text', 99, 3.14, x'0123456789ABCDEF')`)
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	outS, errS, err := s.tc.ExecuteShell([]string{".dump"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	prefix := "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\nCREATE TABLE"
	if !strings.HasSuffix(s.dbUri, "test.sqlite") {
		prefix += " if not exists"
	}
	expected := " alltypes (t text, i integer, r real, b blob);\nINSERT INTO alltypes VALUES('text',99,3.14,X'0123456789ABCDEF');\nCOMMIT;"

	s.tc.AssertSqlEquals(outS, prefix+expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableConainingFieldsWithALLTypes_WhenInsertAndCallDotDumpCommand_ExpectNoErrors() {
	s.tc.CreateAllTypesTable("alltypes", []utils.AllTypesTableEntry{
		{TextNotNullable: "text2", IntNotNullable: 0, FloatNotNullable: 1.5, UnknownNotNullable: 0.0, BlobNotNullable: "0123456789ABCDEF"},
	})

	outS, errS, err := s.tc.ExecuteShell([]string{".dump"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	prefix := "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\nCREATE TABLE"
	if !strings.HasSuffix(s.dbUri, "test.sqlite") {
		prefix += " if not exists"
	}
	expected := " alltypes (textNullable text, textNotNullable text NOT NULL, textWithDefault text DEFAULT 'defaultValue', \n\tintNullable INTEGER, intNotNullable INTEGER NOT NULL, intWithDefault INTEGER DEFAULT '0', \n\tfloatNullable REAL, floatNotNullable REAL NOT NULL, floatWithDefault REAL DEFAULT '0.0', \n\tunknownNullable NUMERIC, unknownNotNullable NUMERIC NOT NULL, unknownWithDefault NUMERIC DEFAULT 0.0, \n\tblobNullable BLOB, blobNotNullable BLOB NOT NULL, blobWithDefault BLOB DEFAULT 'x\"0\"');\nINSERT INTO alltypes VALUES(NULL,'text2','defaultValue',NULL,0,0,NULL,1.5,0,NULL,0,0,NULL,X'0123456789ABCDEF','x\"0\"');\nCOMMIT;"

	s.tc.AssertSqlEquals(outS, prefix+expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableWithRecords_WhenCreateIndexAndCallDotDumpCommand_ExpectNoErrors() {
	s.tc.CreateEmptyAllTypesTable("alltypes")
	_, errS, err := s.tc.Execute("CREATE INDEX idx_textNullable on alltypes (textNullable);CREATE INDEX idx_intNotNullable on alltypes (intNotNullable) WHERE intNotNullable > 1;")
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	outS, errS, err := s.tc.ExecuteShell([]string{".dump"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	prefix := "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\nCREATE TABLE"
	if !strings.HasSuffix(s.dbUri, "test.sqlite") {
		prefix += " if not exists"
	}
	expected := " alltypes (textNullable text, textNotNullable text NOT NULL, textWithDefault text DEFAULT 'defaultValue', \n\tintNullable INTEGER, intNotNullable INTEGER NOT NULL, intWithDefault INTEGER DEFAULT '0', \n\tfloatNullable REAL, floatNotNullable REAL NOT NULL, floatWithDefault REAL DEFAULT '0.0', \n\tunknownNullable NUMERIC, unknownNotNullable NUMERIC NOT NULL, unknownWithDefault NUMERIC DEFAULT 0.0, \n\tblobNullable BLOB, blobNotNullable BLOB NOT NULL, blobWithDefault BLOB DEFAULT 'x\"0\"');\nCREATE INDEX idx_textNullable on alltypes (textNullable);\nCREATE INDEX idx_intNotNullable on alltypes (intNotNullable) WHERE intNotNullable > 1;\nCOMMIT;"

	s.tc.AssertSqlEquals(outS, prefix+expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableWithRecordsWithSingleQuote_WhenCalllDotDumpCommand_ExpectSingleQuoteScape() {
	s.tc.CreateEmptySimpleTable("t")
	_, errS, err := s.tc.Execute("INSERT INTO t VALUES(0, \"x'x\", 0)")
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	outS, errS, err := s.tc.ExecuteShell([]string{".dump"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	prefix := "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\nCREATE TABLE"
	if !strings.HasSuffix(s.dbUri, "test.sqlite") {
		prefix += " if not exists"
	}
	expected := " t (id integer primary key, textfield text, intfield integer);\ninsert into t values(0,'x''x',0);\ncommit;"

	s.tc.AssertSqlEquals(outS, prefix+expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableNameStartingWithNumber_WhenCalllDotDumpCommand_ExpectCorrectFormat() {
	s.tc.CreateSimpleTable("\"8test\"", []utils.SimpleTableEntry{{TextField: "Value", IntField: 1}})

	outS, errS, err := s.tc.ExecuteShell([]string{".dump"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	prefix := "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\nCREATE TABLE"
	if !strings.HasSuffix(s.dbUri, "test.sqlite") {
		prefix += " if not exists"
	}
	expected := " \"8test\" (id integer primary key, textfield text, intfield integer);\ninsert into \"8test\" values(1,'value',1);\ncommit;"

	s.tc.AssertSqlEquals(outS, prefix+expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableNameWithSpecialCharacters_WhenCallDotDumpCommand_ExpectCorrectFormat() {
	s.tc.CreateSimpleTable("\"t+e(s!t?\"", []utils.SimpleTableEntry{{TextField: "Value", IntField: 1}})

	outS, errS, err := s.tc.ExecuteShell([]string{".dump"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	prefix := "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\nCREATE TABLE"
	if !strings.HasSuffix(s.dbUri, "test.sqlite") {
		prefix += " if not exists"
	}
	expected := " \"t+e(s!t?\" (id integer primary key, textfield text, intfield integer);\ninsert into \"t+e(s!t?\" values(1,'value',1);\ncommit;"

	s.tc.AssertSqlEquals(outS, prefix+expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableNameWithTheSameSignatureAsExpainQueryPlan_WhenQueryingIt_ExpectNotToBeTreatedAsExplainQueryPlan() {
	_, _, err := s.tc.Execute("CREATE TABLE fake_explain (ID INTEGER PRIMARY KEY, PARENT INTEGER, NOTUSED INTEGER, DETAIL TEXT);")
	s.tc.Assert(err, qt.IsNil)

	outS, errS, err := s.tc.ExecuteShell([]string{"SELECT * FROM fake_explain;"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	expected := "id     parent     notused     detail"

	s.tc.AssertSqlEquals(outS, expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableWithRecordsWithSingleQuote_WhenCalllSelectAllFromTable_ExpectSingleQuoteScape() {
	s.tc.CreateEmptySimpleTable("t")
	_, errS, err := s.tc.Execute("INSERT INTO t VALUES (0, \"x'x\", 0)")
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	outS, errS, err := s.tc.ExecuteShell([]string{"SELECT * FROM t;"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, "")

	expected := "id     textfield     intfield \n0      x'x           0"

	s.tc.AssertSqlEquals(outS, expected)
}

func (s *DBRootCommandShellSuite) Test_GivenATableWithRecord_WhenCallDotModeCSVAndSelect_ExpectNoErrors() {
	s.tc.CreateSimpleTable("simple_table", []utils.SimpleTableEntry{{TextField: "value \"1", IntField: 1}, {TextField: "value, 2", IntField: 2}})

	outS, errSMode, err := s.tc.ExecuteShell([]string{".mode csv", "SELECT * from simple_table;"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errSMode, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, "id,textField,intField\n1,\"value \"\"1\",1\n2,\"value, 2\",2")
}

func (s *DBRootCommandShellSuite) Test_GivenAnEmptyTable_WhenCallDotModeCSVAndSelect_ExpectNoErrors() {
	s.tc.CreateEmptySimpleTable("simple_table")

	outS, errSMode, err := s.tc.ExecuteShell([]string{".mode csv", "SELECT * from simple_table;"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errSMode, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, "id,textField,intField")
}

func (s *DBRootCommandShellSuite) Test_GivenATableWithRecords_WhenCallDotModeJSONAndSelect_ExpectNoErrors() {
	s.tc.CreateSimpleTable("simple_table", []utils.SimpleTableEntry{{TextField: "value", IntField: 1}, {TextField: "value2", IntField: 2}})

	outS, errSMode, err := s.tc.ExecuteShell([]string{".mode json", "SELECT * from simple_table;"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errSMode, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, `[{"id":"1","intField":"1","textField":"value"},{"id":"2","intField":"2","textField":"value2"}]`)
}

func (s *DBRootCommandShellSuite) Test_GivenAnEmptyTable_WhenCallDotModeJSONAndSelect_ExpectEmptyReturn() {
	s.tc.CreateEmptySimpleTable("simple_table")

	outS, errSMode, err := s.tc.ExecuteShell([]string{".mode json", "SELECT * from simple_table;"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errSMode, qt.Equals, "")
	s.tc.Assert(outS, qt.Equals, "")
}

func (s *DBRootCommandShellSuite) Test_WhenCallACommandThatDoesNotExist_ExpectToReturnAnErrorMessage() {
	outS, errS, err := s.tc.ExecuteShell([]string{".nonExistingCommand"})
	s.tc.Assert(err, qt.IsNil)
	s.tc.Assert(errS, qt.Equals, `Error: unknown command or invalid arguments: ".nonExistingCommand". Enter ".help" for help`)
	s.tc.Assert(outS, qt.Equals, "")
}

func TestDBRootCommandShellSuite_WhenDbIsSQLite(t *testing.T) {
	suite.Run(t, NewDBRootCommandShellSuite(t.TempDir()+"test.sqlite"))
}

func TestDBRootCommandShellSuite_WhenDbIsSqld(t *testing.T) {
	testConfig := utils.GetTestConfig(t)
	if testConfig.SkipSqldTests {
		t.Skip("Skipping SQLD tests due configuration")
	}

	suite.Run(t, NewDBRootCommandShellSuite(testConfig.SqldDbUri))
}
