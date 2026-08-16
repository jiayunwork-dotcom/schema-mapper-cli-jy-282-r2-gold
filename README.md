# schema-mapper-cli

异构数据源 Schema 映射与自动转换命令行工具（纯标准库、离线）。

## 能力

- 解析多种 Schema 输入并统一为内部表示（IR）：
  - JSON Schema（简化 object 形式）
  - SQL DDL（`CREATE TABLE`）
  - CSV 表头推断
- 基于字段名相似度（编辑距离）自动建议映射规则
- 按映射规则把源数据（CSV）转换为目标结构（JSON）
- Schema diff：新增 / 删除 / 类型变更 / 可选性变更
- 兼容性检查：源数据能否无损转换为目标

## 用法

```bash
# 解析一个 schema 文件（按扩展名识别 json/csv/sql）
schema-mapper-cli parse source.sql

# 比较两个 JSON Schema 的差异
schema-mapper-cli diff a.json b.json

# 为两个 schema 建议字段映射规则
schema-mapper-cli suggest source.json target.json

# 按规则把 CSV 转换为目标 JSON
schema-mapper-cli convert -src data.csv -map rules.json -out out.json
```

无参数或未知子命令会打印用法并以非零退出码结束（受控退出，不崩溃）。
