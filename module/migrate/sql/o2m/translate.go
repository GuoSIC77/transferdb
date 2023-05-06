/*
Copyright © 2020 Marvin

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package o2m

import (
	"fmt"
	"github.com/thinkeridea/go-extend/exstrings"
	"github.com/wentaojin/transferdb/common"
	"github.com/wentaojin/transferdb/database/meta"
	"github.com/wentaojin/transferdb/database/mysql"
	"go.uber.org/zap"
	"math"
	"strings"
	"time"
)

// 全量数据导出导入期间，运行安全模式
// INSERT INTO 语句替换成 REPLACE INTO 语句
// 转换表数据 -> 全量任务
func translateTableRecord(
	targetSchemaName, targetTableName, rowidSQL string, columnFields []string, rowsResult []interface{}, insertBatchSize int, safeMode bool) (string, [][]interface{}, string, [][]interface{}) {
	startTime := time.Now()
	columnCounts := len(columnFields)

	// bindVars
	actualBindVarsCounts := len(rowsResult)
	planBindVarsCounts := insertBatchSize * columnCounts

	// 计算可切分数，向下取整
	splitNums := int(math.Floor(float64(actualBindVarsCounts) / float64(planBindVarsCounts)))

	// 计算切分元素在 actualBindVarsCounts 位置
	planIntegerBinds := splitNums * planBindVarsCounts
	// 计算差值
	differenceBinds := actualBindVarsCounts - planIntegerBinds
	// 计算行数
	rowCounts := actualBindVarsCounts / columnCounts

	var (
		prepareSQL1 string
		prepareSQL2 string
		args1       [][]interface{} // batch
		args2       [][]interface{} // single
	)
	if differenceBinds == 0 {
		// batch 写入
		// 切分 batch
		args1 = common.SplitMultipleSlice(rowsResult, int64(splitNums))

		// 计算占位符
		rowBatchCounts := actualBindVarsCounts / columnCounts / splitNums

		prepareSQL1 = common.StringsBuilder(
			GenMySQLInsertSQLStmtPrefix(targetSchemaName, targetTableName, columnFields, safeMode),
			GenMySQLPrepareBindVarStmt(columnCounts, rowBatchCounts))
	} else {
		if planIntegerBinds > 0 {
			// batch 写入
			// 切分 batch
			args1 = common.SplitMultipleSlice(rowsResult[:planIntegerBinds], int64(splitNums))

			// 计算占位符
			rowBatchCounts := planIntegerBinds / columnCounts / splitNums

			prepareSQL1 = common.StringsBuilder(
				GenMySQLInsertSQLStmtPrefix(targetSchemaName, targetTableName, columnFields, safeMode),
				GenMySQLPrepareBindVarStmt(columnCounts, rowBatchCounts))
		}

		// 单次写入
		args2 = append(args2, rowsResult[planIntegerBinds:])
		// 计算占位符
		rowBatchCounts := differenceBinds / columnCounts

		prepareSQL2 = common.StringsBuilder(
			GenMySQLInsertSQLStmtPrefix(targetSchemaName, targetTableName, columnFields, safeMode),
			GenMySQLPrepareBindVarStmt(columnCounts, rowBatchCounts))
	}
	endTime := time.Now()
	zap.L().Info("single full table rowid data translator",
		zap.String("schema", targetSchemaName),
		zap.String("table", targetTableName),
		zap.String("rowid sql", rowidSQL),
		zap.Int("rowid rows", rowCounts),
		zap.Int("insert batch size", insertBatchSize),
		zap.Int("split sql nums", len(args1)+len(args2)),
		zap.Bool("write safe mode", safeMode),
		zap.String("cost", endTime.Sub(startTime).String()))

	return prepareSQL1, args1, prepareSQL2, args2
}

// SQL Prepare 语句
func GenMySQLTablePrepareStmt(
	targetSchemaName, targetTableName string, columnFields []string, insertBatchSize int, safeMode bool) string {
	columnCounts := len(columnFields)

	return common.StringsBuilder(
		GenMySQLInsertSQLStmtPrefix(targetSchemaName, targetTableName, columnFields, safeMode),
		GenMySQLPrepareBindVarStmt(columnCounts, insertBatchSize))
}

// SQL Prefix 语句
func GenMySQLInsertSQLStmtPrefix(targetSchemaName, targetTableName string, columns []string, safeMode bool) string {
	var prefixSQL string
	column := common.StringsBuilder(" (", strings.Join(columns, ","), ")")
	if safeMode {
		prefixSQL = common.StringsBuilder(`REPLACE INTO `, targetSchemaName, ".", targetTableName, column, ` VALUES `)

	} else {
		prefixSQL = common.StringsBuilder(`INSERT INTO `, targetSchemaName, ".", targetTableName, column, ` VALUES `)
	}
	return prefixSQL
}

// SQL Prepare 语句
func GenMySQLPrepareBindVarStmt(columns, bindVarBatch int) string {
	var (
		bindVars []string
		bindVar  []string
	)
	for i := 0; i < columns; i++ {
		bindVar = append(bindVar, "?")
	}
	singleBindVar := common.StringsBuilder("(", exstrings.Join(bindVar, ","), ")")
	for i := 0; i < bindVarBatch; i++ {
		bindVars = append(bindVars, singleBindVar)
	}

	return exstrings.Join(bindVars, ",")
}

// Oracle SQL 转换
// ORACLE 数据库同步需要开附加日志且表需要捕获字段列日志，Logminer 内容 UPDATE/DELETE/INSERT 语句会带所有字段信息
func translateAndAddOracleIncrRecord(dbTypeS, dbTypeT, taskMode, sourceSchema, sourceTable string, metaDB *meta.Meta, mysql *mysql.MySQL, logminers []logminer, taskQueue chan IncrTask) error {

	startTime := time.Now()
	zap.L().Info("oracle table increment log apply start",
		zap.String("oracle schema", sourceSchema),
		zap.String("oracle table", sourceTable),
		zap.Time("start time", startTime))

	for _, rows := range logminers {
		// 如果 sqlRedo 存在记录则继续处理，不存在记录则报错
		if rows.SQLRedo == "" {
			return fmt.Errorf("does not meet expectations [oracle sql redo is be null], please check")
		}

		if rows.Operation == common.MigrateOperationDDL {
			zap.L().Info("translator oracle payload", zap.String("ORACLE DDL", rows.SQLRedo))
		}

		// 移除引号
		rows.SQLRedo = common.ReplaceQuotesString(rows.SQLRedo)
		// 移除分号
		rows.SQLRedo = common.ReplaceSpecifiedString(rows.SQLRedo, ";", "")

		if rows.SQLUndo != "" {
			rows.SQLUndo = common.ReplaceQuotesString(rows.SQLUndo)
			rows.SQLUndo = common.ReplaceSpecifiedString(rows.SQLUndo, ";", "")
			rows.SQLUndo = common.ReplaceSpecifiedString(rows.SQLUndo,
				common.StringsBuilder(rows.SourceSchema, "."),
				common.StringsBuilder(common.StringUPPER(rows.TargetSchema), "."))
		}

		// 比如：INSERT INTO MARVIN.MARVIN1 (ID,NAME) VALUES (1,'marvin')
		// 比如：DELETE FROM MARVIN.MARVIN7 WHERE ID = 5 and NAME = 'pyt'
		// 比如：UPDATE MARVIN.MARVIN1 SET ID = 2 , NAME = 'marvin' WHERE ID = 2 AND NAME = 'pty'
		// 比如: drop table marvin.marvin7
		// 比如: truncate table marvin.marvin7
		mysqlRedo, operationType, err := translateOracleToMySQLSQL(rows.SQLRedo, rows.SQLUndo, common.StringUPPER(rows.TargetSchema), common.StringUPPER(rows.TargetTable))
		if err != nil {
			return err
		}

		// 注册任务到 Job 队列
		lp := IncrTask{
			Ctx:            mysql.Ctx,
			DBTypeS:        dbTypeS,
			DBTypeT:        dbTypeT,
			TaskMode:       taskMode,
			MetaDB:         metaDB,
			MySQL:          mysql,
			GlobalSCN:      rows.SCN, // 更新元数据 GLOBAL_SCN 至当前消费的 SCN 号
			SourceTableSCN: rows.SCN,
			SourceSchema:   rows.SourceSchema,
			SourceTable:    rows.SourceTable,
			TargetSchema:   rows.TargetSchema,
			TargetTable:    rows.TargetTable,
			OracleRedo:     rows.SQLRedo,
			MySQLRedo:      mysqlRedo,
			Operation:      rows.Operation,
			OperationType:  operationType}

		// 避免太多日志输出
		// zlog.zap.L().Info("translator oracle payload", zap.String("payload", lp.Marshal()))
		taskQueue <- lp
	}

	endTime := time.Now()
	zap.L().Info("oracle table increment log apply finished",
		zap.String("oracle schema", sourceSchema),
		zap.String("oracle table", sourceTable),
		zap.String("status", "success"),
		zap.Time("start time", startTime),
		zap.Time("end time", endTime),
		zap.String("cost time", time.Since(startTime).String()))

	// 任务结束，关闭通道
	close(taskQueue)

	return nil
}

// Oracle SQL 转换
// 1、INSERT INTO / REPLACE INTO
// 2、UPDATE / DELETE、REPLACE INTO
func translateOracleToMySQLSQL(oracleSQLRedo, oracleSQLUndo, targetSchema, targetTable string) ([]string, string, error) {
	var (
		sqls          []string
		operationType string
	)
	astNode, err := parseSQL(oracleSQLRedo)
	if err != nil {
		return []string{}, operationType, fmt.Errorf("parse error: %v\n", err.Error())
	}

	stmt := extractStmt(astNode)

	// 库名、表名转换
	stmt.Schema = targetSchema
	stmt.Table = targetTable

	switch {
	case stmt.Operation == common.MigrateOperationUpdate:
		operationType = common.MigrateOperationUpdate
		astUndoNode, err := parseSQL(oracleSQLUndo)
		if err != nil {
			return []string{}, operationType, fmt.Errorf("parse error: %v\n", err.Error())
		}
		undoStmt := extractStmt(astUndoNode)

		stmt.Data = undoStmt.Before
		for column, _ := range stmt.Before {
			stmt.Columns = append(stmt.Columns, strings.ToUpper(column))
		}

		var deleteSQL string

		if stmt.WhereExpr == "" {
			deleteSQL = common.StringsBuilder(`DELETE FROM `, stmt.Schema, ".", stmt.Table)
		} else {
			deleteSQL = common.StringsBuilder(`DELETE FROM `, stmt.Schema, ".", stmt.Table, ` `, stmt.WhereExpr)
		}

		var (
			values []string
		)
		for _, col := range stmt.Columns {
			values = append(values, stmt.Data[col].(string))
		}
		insertSQL := common.StringsBuilder(`REPLACE INTO `, stmt.Schema, ".", stmt.Table,
			"(",
			strings.Join(stmt.Columns, ","),
			")",
			` VALUES `,
			"(",
			strings.Join(values, ","),
			")")

		sqls = append(sqls, deleteSQL)
		sqls = append(sqls, insertSQL)

	case stmt.Operation == common.MigrateOperationInsert:
		operationType = common.MigrateOperationInsert

		var values []string

		for _, col := range stmt.Columns {
			values = append(values, stmt.Data[col].(string))
		}
		replaceSQL := common.StringsBuilder(`REPLACE INTO `, stmt.Schema, ".", stmt.Table,
			"(",
			strings.Join(stmt.Columns, ","),
			")",
			` VALUES `,
			"(",
			strings.Join(values, ","),
			")")

		sqls = append(sqls, replaceSQL)

	case stmt.Operation == common.MigrateOperationDelete:
		operationType = common.MigrateOperationDelete

		var deleteSQL string

		if stmt.WhereExpr == "" {
			deleteSQL = common.StringsBuilder(`DELETE FROM `, stmt.Schema, ".", stmt.Table)
		} else {
			deleteSQL = common.StringsBuilder(`DELETE FROM `, stmt.Schema, ".", stmt.Table, ` `, stmt.WhereExpr)
		}

		sqls = append(sqls, deleteSQL)

	case stmt.Operation == common.MigrateOperationTruncate:
		operationType = common.MigrateOperationTruncateTable

		truncateSQL := common.StringsBuilder(`TRUNCATE TABLE `, stmt.Schema, ".", stmt.Table)
		sqls = append(sqls, truncateSQL)

	case stmt.Operation == common.MigrateOperationDrop:
		operationType = common.MigrateOperationDropTable

		dropSQL := common.StringsBuilder(`DROP TABLE `, stmt.Schema, ".", stmt.Table)

		sqls = append(sqls, dropSQL)
	}
	return sqls, operationType, nil
}
