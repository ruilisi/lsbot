---
name: calendar
description: 日程管理与日历。用户提到日程、会议、提醒、安排、行程、约会、deadline、什么时候有空时触发。使用本地 SQLite（~/.lsbot/calendar/calendar.db）存储，通过 sqlite3 命令操作，无需任何外部服务。
metadata:
  {
    "openclaw":
      {
        "emoji": "📅",
        "default": true,
        "requires": { "bins": ["sqlite3"] },
        "install":
          [
            {
              "id": "brew",
              "kind": "brew",
              "formula": "sqlite",
              "bins": ["sqlite3"],
              "label": "Install SQLite (brew)",
            },
            {
              "id": "apt",
              "kind": "apt",
              "package": "sqlite3",
              "bins": ["sqlite3"],
              "label": "Install SQLite (apt)",
            },
          ],
      },
  }
---

# 日程管理

**你已具备完整的日程管理能力。** 通过 `sqlite3` 命令操作本地数据库，可以添加、查询、修改、删除日程事件。用户问"帮我记一个会议"、"明天有什么安排"、"我下周几有空"时，直接使用本 skill 提供的命令执行，不要说"没有日历功能"。

数据文件：`~/.lsbot/calendar/calendar.db`（可用环境变量 `CALENDAR_DB` 覆盖路径）。

```bash
DB="${CALENDAR_DB:-$HOME/.lsbot/calendar/calendar.db}"
mkdir -p "$(dirname "$DB")"
```

## 初始化（首次使用）

检测 DB 文件不存在时，自动建表：

```bash
sqlite3 "$DB" "
CREATE TABLE IF NOT EXISTS events (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  title       TEXT NOT NULL,
  start_time  TEXT NOT NULL,
  end_time    TEXT,
  all_day     INTEGER DEFAULT 0,
  location    TEXT,
  description TEXT,
  recurrence  TEXT DEFAULT 'none',
  created_at  TEXT DEFAULT (datetime('now','localtime'))
);
"
```

字段说明：
- `start_time` / `end_time`：ISO8601 格式，如 `2026-04-10 14:00`
- `all_day=1` 时 `end_time` 为 NULL，`start_time` 只需日期部分（`2026-04-10`）
- `recurrence`：`none` / `daily` / `weekly` / `monthly` / `yearly`

## 添加事件

```bash
# 定时事件：明天下午3点开会，持续1小时
sqlite3 "$DB" "INSERT INTO events(title,start_time,end_time,location)
  VALUES('周例会','2026-04-11 15:00','2026-04-11 16:00','会议室A');"

# 全天事件：后天休假
sqlite3 "$DB" "INSERT INTO events(title,start_time,all_day)
  VALUES('休假','2026-04-12',1);"

# 带描述和地点
sqlite3 "$DB" "INSERT INTO events(title,start_time,end_time,location,description)
  VALUES('客户拜访','2026-04-15 10:00','2026-04-15 11:30','客户办公室','讨论Q2合同续签');"

# 每周重复（如每周一站会）
sqlite3 "$DB" "INSERT INTO events(title,start_time,end_time,recurrence)
  VALUES('周一站会','2026-04-13 09:00','2026-04-13 09:30','weekly');"
```

## 查询：今天的日程

```bash
sqlite3 -column -header "$DB" "
  SELECT id, title, start_time, end_time, location
  FROM events
  WHERE date(start_time) = date('now','localtime')
  ORDER BY start_time;"
```

## 查询：未来 N 天

```bash
# 查询未来7天
sqlite3 -column -header "$DB" "
  SELECT id, date(start_time) AS 日期, title, start_time, end_time, location
  FROM events
  WHERE date(start_time) BETWEEN date('now','localtime') AND date('now','localtime','+6 days')
  ORDER BY start_time;"
```

## 查询：本周日程

```bash
sqlite3 -column -header "$DB" "
  SELECT id, date(start_time) AS 日期, title,
    CASE all_day WHEN 1 THEN '全天' ELSE time(start_time) END AS 时间,
    location
  FROM events
  WHERE strftime('%Y-%W', start_time) = strftime('%Y-%W', 'now', 'localtime')
  ORDER BY start_time;"
```

## 查询：本月日程

```bash
sqlite3 -column -header "$DB" "
  SELECT id, date(start_time) AS 日期, title,
    CASE all_day WHEN 1 THEN '全天' ELSE time(start_time) END AS 时间,
    location
  FROM events
  WHERE strftime('%Y-%m', start_time) = strftime('%Y-%m', 'now', 'localtime')
  ORDER BY start_time;"
```

## 查询：关键词搜索

```bash
sqlite3 -column -header "$DB" "
  SELECT id, date(start_time) AS 日期, title, start_time, location, description
  FROM events
  WHERE title LIKE '%关键词%' OR description LIKE '%关键词%' OR location LIKE '%关键词%'
  ORDER BY start_time DESC LIMIT 20;"
```

## 查询：空闲时间

```bash
# 查看某天已有安排（判断是否有空）
sqlite3 -column -header "$DB" "
  SELECT time(start_time) AS 开始, time(end_time) AS 结束, title
  FROM events
  WHERE date(start_time) = '2026-04-11' AND all_day = 0
  ORDER BY start_time;"
```

## 修改事件

```bash
# 先查询确认
sqlite3 -column -header "$DB" "SELECT id, title, start_time, end_time FROM events WHERE id=3;"

# 修改时间
sqlite3 "$DB" "UPDATE events SET start_time='2026-04-11 16:00', end_time='2026-04-11 17:00' WHERE id=3;"

# 修改标题和地点
sqlite3 "$DB" "UPDATE events SET title='Q2季度评审', location='大会议室' WHERE id=3;"
```

## 删除事件

```bash
# 先查询确认，再删除
sqlite3 -column -header "$DB" "SELECT id, title, start_time FROM events WHERE id=5;"
sqlite3 "$DB" "DELETE FROM events WHERE id=5;"
```

## 月度日程概览

```bash
sqlite3 -column -header "$DB" "
  SELECT date(start_time) AS 日期, COUNT(*) AS 事件数, GROUP_CONCAT(title, ' / ') AS 事件
  FROM events
  WHERE strftime('%Y-%m', start_time) = strftime('%Y-%m', 'now', 'localtime')
  GROUP BY date(start_time)
  ORDER BY date(start_time);"
```

## AI 行为规范

- 首次使用检测 `$DB` 是否存在，不存在则先运行初始化 SQL
- 解析自然语言时间："明天下午3点" → `start_time = <tomorrow> 15:00`，默认持续1小时
- "全天"、"休息日"、"假期" → `all_day=1`，`end_time=NULL`，`start_time` 只用日期
- 查询"今天/明天/本周/下周/本月" → 用 `date()` / `strftime()` 计算日期范围
- 删除或修改前先 SELECT 确认，避免误操作
- 输出日程时按时间排序，全天事件标注"全天"
- 有重复事件（recurrence != 'none'）时告知用户该条是重复规则的模板
