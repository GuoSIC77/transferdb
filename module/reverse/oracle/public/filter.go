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
package public

import (
	"fmt"
	"github.com/wentaojin/transferdb/common"
	"github.com/wentaojin/transferdb/config"
	"github.com/wentaojin/transferdb/database/oracle"
	"github.com/wentaojin/transferdb/filter"
	"go.uber.org/zap"
	"time"
)

func FilterCFGTable(cfg *config.Config, oracle *oracle.Oracle) ([]string, error) {
	startTime := time.Now()
	var (
		exporterTableSlice []string
		excludeTables      []string
		err                error
	)

	// 获取 oracle 所有 schema
	allOraSchemas, err := oracle.GetOracleSchemas()
	if err != nil {
		return nil, err
	}

	if !common.IsContainString(allOraSchemas, common.StringUPPER(cfg.OracleConfig.SchemaName)) {
		return nil, fmt.Errorf("oracle schema [%s] isn't exist in the database", cfg.OracleConfig.SchemaName)
	}

	// 获取 oracle 所有数据表
	allTables, err := oracle.GetOracleSchemaTable(common.StringUPPER(cfg.OracleConfig.SchemaName))
	if err != nil {
		return exporterTableSlice, err
	}

	switch {
	case len(cfg.OracleConfig.IncludeTable) != 0 && len(cfg.OracleConfig.ExcludeTable) == 0:
		// 过滤规则加载
		f, err := filter.Parse(cfg.OracleConfig.IncludeTable)
		if err != nil {
			panic(err)
		}

		for _, t := range allTables {
			if f.MatchTable(t) {
				exporterTableSlice = append(exporterTableSlice, t)
			}
		}
	case len(cfg.OracleConfig.IncludeTable) == 0 && len(cfg.OracleConfig.ExcludeTable) != 0:
		// 过滤规则加载
		f, err := filter.Parse(cfg.OracleConfig.ExcludeTable)
		if err != nil {
			panic(err)
		}

		for _, t := range allTables {
			if f.MatchTable(t) {
				excludeTables = append(excludeTables, t)
			}
		}
		exporterTableSlice = common.FilterDifferenceStringItems(allTables, excludeTables)

	case len(cfg.OracleConfig.IncludeTable) == 0 && len(cfg.OracleConfig.ExcludeTable) == 0:
		exporterTableSlice = allTables

	default:
		return exporterTableSlice, fmt.Errorf("source config params include-table/exclude-table cannot exist at the same time")
	}

	if len(exporterTableSlice) == 0 {
		return exporterTableSlice, fmt.Errorf("exporter tables aren't exist, please check config params include-table/exclude-table")
	}

	endTime := time.Now()
	zap.L().Info("get oracle to mysql all tables",
		zap.String("schema", cfg.OracleConfig.SchemaName),
		zap.Strings("exporter tables list", exporterTableSlice),
		zap.Int("include table counts", len(exporterTableSlice)),
		zap.Int("exclude table counts", len(excludeTables)),
		zap.Int("all table counts", len(allTables)),
		zap.String("cost", endTime.Sub(startTime).String()))

	return exporterTableSlice, nil
}

func FilterOracleCompatibleTable(cfg *config.Config, oracle *oracle.Oracle, exporters []string) ([]string, []string, []string, []string, []string, error) {
	partitionTables, err := filterOraclePartitionTable(cfg, oracle, exporters)
	if err != nil {
		return []string{}, []string{}, []string{}, []string{}, []string{}, fmt.Errorf("error on filter r.Oracle partition table: %v", err)
	}
	temporaryTables, err := filterOracleTemporaryTable(cfg, oracle, exporters)
	if err != nil {
		return []string{}, []string{}, []string{}, []string{}, []string{}, fmt.Errorf("error on filter r.Oracle temporary table: %v", err)

	}
	clusteredTables, err := filterOracleClusteredTable(cfg, oracle, exporters)
	if err != nil {
		return []string{}, []string{}, []string{}, []string{}, []string{}, fmt.Errorf("error on filter r.Oracle clustered table: %v", err)

	}
	materializedView, err := filterOracleMaterializedView(cfg, oracle, exporters)
	if err != nil {
		return []string{}, []string{}, []string{}, []string{}, []string{}, fmt.Errorf("error on filter r.Oracle materialized view: %v", err)

	}

	if len(partitionTables) != 0 {
		zap.L().Warn("partition tables",
			zap.String("schema", cfg.OracleConfig.SchemaName),
			zap.String("partition table list", fmt.Sprintf("%v", partitionTables)),
			zap.String("suggest", "if necessary, please manually convert and process the tables in the above list"))
	}
	if len(temporaryTables) != 0 {
		zap.L().Warn("temporary tables",
			zap.String("schema", cfg.OracleConfig.SchemaName),
			zap.String("temporary table list", fmt.Sprintf("%v", temporaryTables)),
			zap.String("suggest", "if necessary, please manually process the tables in the above list"))
	}
	if len(clusteredTables) != 0 {
		zap.L().Warn("clustered tables",
			zap.String("schema", cfg.OracleConfig.SchemaName),
			zap.String("clustered table list", fmt.Sprintf("%v", clusteredTables)),
			zap.String("suggest", "if necessary, please manually process the tables in the above list"))
	}

	var exporterTables []string
	if len(materializedView) != 0 {
		zap.L().Warn("materialized views",
			zap.String("schema", cfg.OracleConfig.SchemaName),
			zap.String("materialized view list", fmt.Sprintf("%v", materializedView)),
			zap.String("suggest", "if necessary, please manually process the tables in the above list"))

		// 排除物化视图
		exporterTables = common.FilterDifferenceStringItems(exporters, materializedView)
	} else {
		exporterTables = exporters
	}
	return partitionTables, temporaryTables, clusteredTables, materializedView, exporterTables, nil
}

func filterOraclePartitionTable(cfg *config.Config, oracle *oracle.Oracle, exporters []string) ([]string, error) {
	tables, err := oracle.GetOracleSchemaPartitionTable(common.StringUPPER(cfg.OracleConfig.SchemaName))
	if err != nil {
		return nil, err
	}
	return common.FilterIntersectionStringItems(exporters, tables), nil
}

func filterOracleTemporaryTable(cfg *config.Config, oracle *oracle.Oracle, exporters []string) ([]string, error) {
	tables, err := oracle.GetOracleSchemaTemporaryTable(common.StringUPPER(cfg.OracleConfig.SchemaName))
	if err != nil {
		return nil, err
	}
	return common.FilterIntersectionStringItems(exporters, tables), nil
}

func filterOracleClusteredTable(cfg *config.Config, oracle *oracle.Oracle, exporters []string) ([]string, error) {
	tables, err := oracle.GetOracleSchemaClusteredTable(common.StringUPPER(cfg.OracleConfig.SchemaName))
	if err != nil {
		return nil, err
	}
	return common.FilterIntersectionStringItems(exporters, tables), nil
}

func filterOracleMaterializedView(cfg *config.Config, oracle *oracle.Oracle, exporters []string) ([]string, error) {
	tables, err := oracle.GetOracleSchemaMaterializedView(common.StringUPPER(cfg.OracleConfig.SchemaName))
	if err != nil {
		return nil, err
	}
	return common.FilterIntersectionStringItems(exporters, tables), nil
}
