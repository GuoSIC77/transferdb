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
package meta

import (
	"context"
	"fmt"
	"github.com/wentaojin/transferdb/common"
	"gorm.io/gorm"
)

// 自定义库转换规则 - schema 级别
type SchemaDatatypeRule struct {
	ID          uint   `gorm:"primary_key;autoIncrement;comment:'自增编号'" json:"id"`
	DBTypeS     string `gorm:"type:varchar(15);index:idx_dbtype_st_map,unique;comment:'源数据库类型'" json:"db_type_s"`
	DBTypeT     string `gorm:"type:varchar(15);index:idx_dbtype_st_map,unique;comment:'目标数据库类型'" json:"db_type_t"`
	SchemaNameS string `gorm:"type:varchar(200);not null;index:idx_dbtype_st_map,unique;comment:'源端库 schema'" json:"schema_name_s"`
	ColumnTypeS string `gorm:"type:varchar(300);not null;index:idx_dbtype_st_map,unique;comment:'源端表字段类型'" json:"column_type_s"`
	ColumnTypeT string `gorm:"not null;comment:'目标表字段类型'" json:"column_type_t"`
	*BaseModel
}

func NewSchemaDatatypeRuleModel(m *Meta) *SchemaDatatypeRule {
	return &SchemaDatatypeRule{BaseModel: &BaseModel{
		Meta: m,
	}}
}

func (rw *SchemaDatatypeRule) ParseSchemaTable() (string, error) {
	stmt := &gorm.Statement{DB: rw.GormDB}
	err := stmt.Parse(rw)
	if err != nil {
		return "", fmt.Errorf("parse struct [SchemaDatatypeRule] get table_name failed: %v", err)
	}
	return stmt.Schema.Table, nil
}

func (rw *SchemaDatatypeRule) DetailSchemaRule(ctx context.Context, detailS *SchemaDatatypeRule) ([]SchemaDatatypeRule, error) {
	var schemaRuleMap []SchemaDatatypeRule

	table, err := rw.ParseSchemaTable()
	if err != nil {
		return nil, err
	}
	if err = rw.DB(ctx).Where("UPPER(db_type_s) = ? AND UPPER(db_type_t) = ? AND UPPER(schema_name_s) = ?",
		common.StringUPPER(detailS.DBTypeS),
		common.StringUPPER(detailS.DBTypeT),
		common.StringUPPER(detailS.SchemaNameS)).Find(&schemaRuleMap).Error; err != nil {
		return schemaRuleMap, fmt.Errorf("detail table [%s] record failed: %v", table, err)
	}
	return schemaRuleMap, nil
}
