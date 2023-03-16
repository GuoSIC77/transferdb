TransferDB 使用手册
-------
#### 使用说明
1. 表结构定义转换
   - O2M
      1. 常规表定义 reverse_${sourcedb}.sql 文件
      2. 不兼容性对象 compatibility_${sourcedb}.sql 文件【外键、检查约束、分区表、索引等不兼容对象】
      3. 自定义配置表字段规则映射
         1. 数据类型自定义 【column -> table -> schema -> 内置】
            - 库级别数据类型自定义
            - 表级别数据类型自定义
            - 字段级别数据类型自定义
      4. 默认值自定义【global 全局级别】
         1. 任何 schema/table 转换都需要，内置 sysdate -> now() 转换规则
         2. 任何 schema/table 转换都需要，内置 sys_guid() -> uuid() 转换规则
      5. 内置数据类型规则映射，[内置数据类型映射规则](buildin_rule_reverse_o.md)
      6. 表索引定义转换
      7. 表非空约束、外键约束、检查约束、主键约束、唯一约束转换，主键、唯一、检查、外键等约束 ORACLE ENABLED 状态才会被创建，其他状态忽略创建
      8. 注意事项
         1. 分区表统一视为普通表转换，对象输出到 compatibility_${sourcedb}.sql 文件并提供 WARN 日志关键字筛选打印，若有要求，建议 reverse 手工转换
         2. 临时表统一视为普通表转换，对象输出到 compatibility_${sourcedb}.sql 文件并提供 WARN 日志关键字筛选打印
         3. 蔟表统一视为普通表转换，对象输出到 compatibility_${sourcedb}.sql 文件并提供 WARN 日志关键字筛选打印
         4. ORACLE 物化视图不转换，对象输出到 compatibility_${sourcedb}.sql 文件并提供 WARN 日志关键字筛选打印
         5. ORACLE 唯一约束基于唯一索引的字段，下游只会创建唯一索引
         6. ORACLE 字段函数默认值保持上游值，若是下游不支持的默认值，则当手工执行表创建脚本报错
         7. ORACLE FUNCTION-BASED NORMAL、BITMAP 不兼容性索引对象输出到 compatibility_${sourcedb}.sql 文件，并提供 WARN 日志关键字筛选打印
         8. 表结构以及 Schema 定义转换忽略 Oracle 字符集统一以 utf8mb4 转换，但排序规则会根据 Oracle 排序规则予以规则转换
         9. 程序 reverse 阶段若遇到报错则进程不终止，日志最后会输出警告信息，具体错误表以及对应错误详情见 {元数据库} 内表 [error_log_detail] 数据
   - M2O
      1. 常规表定义 reverse_${sourcedb}.sql 文件
      2. 不兼容性对象 compatibility_${sourcedb}.sql 文件【数据类型 ENUM、SET、BIT 等不兼容对象】
      3. 自定义配置表字段规则映射
         1. 数据类型自定义 【column -> table -> schema -> 内置】
            - 库级别数据类型自定义
            - 表级别数据类型自定义
            - 字段级别数据类型自定义
      4. 默认值自定义【global 全局级别】
         1. 任何 schema/table 转换都需要，内置 now() -> sysdate 转换规则
      5. 内置数据类型规则映射，[内置数据类型映射规则](buildin_rule_reverse_m.md)
      6. 表索引定义转换
      7. 表非空约束、外键约束、检查约束、主键约束、唯一约束转换
      8. 注意事项
         1. MySQL 字段函数默认值保持上游值，若是下游不支持的默认值，则当手工执行表创建脚本报错
         2. 表结构以及 Schema 定义转换忽略 MySQL 字符集统一以 AL32UTF8 转换，但 ORACLE 12.2 版本及以上排序规则会根据 MySQL 排序规则予以规则转换，其他 ORACLE 版本若 MySQL 表与字段排序规则不一致则输出到不兼容性文件 compatibility_${sourcedb}.sql
         3. TiDB 临时表统一视为普通表转换，需要人工识别转换
         4. View 视图会输出到兼容性文件 compatibility_${sourcedb}.sql
         5. MySQL/TiDB 字段默认值系统视图，未区分数值、字符类型，不统一，比如：对于字符串默认值 1，显示 1，字符串默认值不会自动加单引号，函数 CURRENT_TIMESTAMP 未加括号，当前默认处理 CURRENT_TIMESTAMP 不加单引号，字符串默认值正则未匹配到()，统一视作字符串，自动加单引号
         6. 程序 reverse 阶段若遇到报错则进程不终止，日志最后会输出警告信息，具体错误表以及对应错误详情见 {元数据库} 内表 [error_log_detail] 数据
2. 表结构对比【以 ORACLE 为基准】
   1. 表结构对比以 ORACLE 为基准对比
      1. 若上下游对比不一致，对比详情以及相关修复 SQL 语句输出 check_${sourcedb}.sql 文件
      2. 若上游字段数少，下游字段数多会自动生成删除 SQL 语句
      3. 若上游字段数多，下游字段数少会自动生成创建 SQL 语句
   2. 注意事项
      1. 表数据类型对比以 TransferDB 内置转换规则为基准，若下游表数据类型与基准不符则输出 
      2. 索引对比会忽略索引名对比，依据索引类型直接对比索引字段是否存在，解决上下游不同索引名，同个索引字段检查不一致问题
      3. ORACLE 字符数据类型 Char / Bytes ，默认 Bytes，MySQL/TiDB 是字符长度，TransferDB 只有当 Scale 数值不一致时才输出不一致
      4. 字符集检查（only 表），匹配转换 Oracle AL32UTF8 -> UTF8MB4/ ZHS16GBK -> GBK 检查，ORACLE GBK 统一视作 UTF8MB4 检查，其他暂不支持检查
      5. 排序规则检查（only 表以及字段列），ORACLE 12.2 及以上版本按字段、表维度匹配转换检查，ORACLE 12.2 以下版本按 DB 维度匹配转换检查
      6. TiDB 数据库排除外键、检查约束对比，MySQL 低版本只检查外键约束，高版本外键、检查约束都对比
      7. MySQL/TiDB timestamp 类型只支持精度 6，oracle 精度最大是 9，会检查出来但是保持原样
      8. 程序 check 阶段若遇到报错则进程不终止，日志最后会输出警告信息，具体错误表以及对应错误详情见 {元数据库} 内表 [error_log_detail] 数据

3. 对象信息收集
   1. 收集现有 ORACLE 数据库内表、索引、分区表、字段长度等信息，输出类似 AWR 报告 report_${sourcedb}.html 文件，用于评估迁移至 MySQL/TiDB 成本

4. 数据同步【ORACLE 11g 及以上版本】 
   1. 数据同步需要存在主键或者唯一键
   2. 数据同步无论 FULL / ALL 模式需要注意时间格式，ORACLE date 格式复杂，同步前可先简单验证下迁移时间格式是否存在问题，transferdb timezone PICK 数据库操作系统的时区
   3. FULL 模式【全量数据导出导入】
      1. 数据同步导出导入要求表存在主键或者唯一键，否则因异常错误退出或者手工中断退出，断点续传【replace into】无法替换，数据可能会导致重复【除非手工清理下游重新导入】
      2. 注意事项：
         - 断点续传期间，配置文件可能涉及迁移表变更的配置不得更改，否则会因迁移表数不一致，而自动判定无法断点续传
         - 断点续传失败，可通过配置 enable-checkpoint = false 自动清理断点以及已迁移的表数据，重新导出导入或者手工清理下游元数据库记录重新导出导入
   4. ALL 模式【全量导出导入 + 增量数据同步】
      1. 增量基于 logminer 日志数据同步，存在 logminer 同等限制，且只同步 INSERT/DELETE/UPDATE DML 以及 DROP TABLE/TRUNCATE TABLE DDL，执行过 TRUNCATE TABLE/ DROP TABLE 可能需要重新增加表附加日志
      2. 基于 logminer 日志数据同步，挖掘速率取决于重做日志磁盘+归档日志磁盘【若在归档日志中】以及 PGA 内存
      3. ALL 模式同步权限以及要求详情见下【ALL 模式同步】

5. CSV 文件数据导出【ORACLE 11g 及以上版本】

6. 数据校验【ORACLE 11g 及以上版本】
   1. 数据校验以及表结构校验以上游 ORACLE 数据库为基准，上游数据存在，下游不存在则新增，下游数据存在，上游数据不存在则删除，输出文件以参数配置 fix-sql-file 命名
   2. 数据校验上游 ORACLE 数据库（主键/唯一键/唯一索引 + NUMBER 类型字段）
      1. 表必须带有主键/唯一键/唯一索引，可以是任意类型的，否则可能出现数据对比不准，如果表不存在主键或唯一键则预检查直接报错中断
      2. 表必须带有 NUMBER 类型字段，NUMBER 类型字段可以是主键、唯一键、唯一索引、普通索引、联合索引
            1. NUMBER 类型字段优先选用单列主键/唯一建/唯一索引，其次选用 DISTINCT 数值高的普通索引或者前导列是 NUMBER 类型的字段
            2. 如果未配置 where 且表 pk/uk/index 不存在 number 字段则预检查直接报错中断
   3. 可选只对比数据行数 VS 对比详情产生修复文件，只对比数据行将不会输出详情修复文件
   4. 可选自定义某张表自定义 range/index-fields 参数配置
      1. 配置文件参数 range 优先级高于 index-fields，仅当两个都配置时，以 range 为准且忽略是否存在索引
   5. 可选断点续传
      1. 断点续传期间，配置文件可能涉及迁移表变更的配置不得更改，否则会因迁移表数不一致，而自动判定无法断点续传 
      2. 断点续传失败，可通过配置 enable-checkpoint = false 自动清理断点，重新数据校验对比
   6. 除预检查阶段外，程序 diff 数据校验阶段若遇到报错则进程不终止，日志最后会输出警告信息，具体错误表以及对应错误详情见 {元数据库} 内表 [error_log_detail] 数据

#### 使用事项

```
1、下载 oracle client，参考官网下载地址 https://www.oracle.com/database/technologies/instant-client/linux-x86-64-downloads.html

2、上传 oracle client 至程序运行服务器，并解压到指定目录，比如：/data1/soft/client/instantclient_19_8

3、配置 transferdb config.toml 参数文件, oracle instance client 参数 lib-dir
lib-dir = "/data1/soft/client/instantclient_19_8"

4、配置 transferdb 参数文件，config.toml 相关参数配置说明见 conf/config.toml

5、表结构转换，[输出示例](example/reverse_${sourcedb}.sql 以及 example/compatibility_${sourcedb}.sql)
$ ./transferdb -config config.toml -mode prepare
$ ./transferdb -config config.toml -mode reverse -source oracle -target mysql
$ ./transferdb -config config.toml -mode reverse -source mysql -target oracle

元数据库[默认 transferdb]自定义转换规则，规则优先级【字段 -> 表 -> 库 -> 内置】
文件自定义规则示例：
表 [schema_datatype_rule] 用于库级别自定义转换规则，库级别优先级高于内置规则
表 [table_datatype_rule]  用于表级别自定义转换规则，表级别优先级高于库级别、高于内置规则
表 [column_datatype_rule] 用于字段级别自定义转换规则，字段级别优先级高于表级别、高于库级别、高于内置规则
表 [buildin_global_defaultval] 用于字段默认值自定义转换规则，优先级适用于全局，注意：自定义默认值是字符 character 数据时需要带有单引号
表 [buildin_column_defaultval] 用于字段默认值自定义转换规则，优先级适用于表级别字段，注意：自定义默认值字符 character 数据时需要带有单引号
insert into buildin_column_defaultval (db_type_s,db_type_t,schema_name_s,table_name_s,column_name_s,default_value_s,default_value_t) values('ORACLE','MYSQL','MARVIN','REVERSE_TIMS01','V1','''marvin01''','''marvin02''');


6、表结构检查(独立于表结构转换，可单独运行，校验规则使用内置规则，[输出示例](example/check_${sourcedb}.sql)
$ ./transferdb -config config.toml -mode prepare
$ ./transferdb -config config.toml -mode check -source oracle -target mysql

7、收集现有 Oracle 数据库内表、索引、分区表、字段长度等信息用于评估迁移成本，[输出示例](example/report_marvin.html)
$ ./transferdb -config config.toml -mode assess -source oracle -target mysql

8、数据全量抽数
$ ./transferdb -config config.toml -mode full -source oracle -target mysql

9、数据同步（全量 + 增量）
$ ./transferdb -config config.toml -mode all -source oracle -target mysql

10、CSV 文件数据导出
$ ./transferdb -config config.toml -mode csv -source oracle -target mysql

11、数据校验，[输出示例](example/fix.sql)
$ ./transferdb -config config.toml -mode prepare
$ ./transferdb -config config.toml -mode compare -source oracle -target mysql
```

#### 程序运行
直接在命令行中用 `nohup` 启动程序，可能会因为 SIGHUP 信号而退出，建议把 `nohup` 放到脚本里面且不建议用 kill -9，如：

```shell
#!/bin/bash
nohup ./transferdb -config config.toml -mode all -source oracle -target mysql > nohup.out &
```