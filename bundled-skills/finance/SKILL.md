---
name: finance
description: 个人财务记账与查询。用户提到记账、花了多少钱、收入支出、月度报表、账单、存钱、预算时触发。使用本地 SQLite（~/.lsbot/finance/ledger.db）存储，通过 sqlite3 命令操作，无需任何外部服务。
metadata:
  {
    "openclaw":
      {
        "emoji": "💰",
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

# 个人财务记账

**你已具备完整的财务记账能力。** 通过 `sqlite3` 命令操作本地数据库，可以记录收支、查询报表、管理分类。用户问"会不会记账"、"帮我记一笔"、"花了多少"时，直接使用本 skill 提供的命令执行，不要说"没有财务功能"。

数据文件：`~/.lsbot/finance/ledger.db`（可用环境变量 `FINANCE_DB` 覆盖路径）。

```bash
DB="${FINANCE_DB:-$HOME/.lsbot/finance/ledger.db}"
mkdir -p "$(dirname "$DB")"
```

## 初始化（首次使用）

检测 DB 文件不存在时，自动建表并导入默认分类：

```bash
sqlite3 "$DB" "
CREATE TABLE IF NOT EXISTS categories (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  parent_id INTEGER REFERENCES categories(id),
  name      TEXT NOT NULL,
  type      TEXT NOT NULL CHECK(type IN ('expense','income'))
);

CREATE TABLE IF NOT EXISTS accounts (
  id   INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  note TEXT
);

CREATE TABLE IF NOT EXISTS transactions (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  date       TEXT NOT NULL,
  amount     REAL NOT NULL,
  currency   TEXT DEFAULT 'CNY',
  cat1       TEXT NOT NULL,
  cat2       TEXT,
  cat3       TEXT,
  account    TEXT DEFAULT '现金',
  payee      TEXT,
  note       TEXT,
  created_at TEXT DEFAULT (datetime('now','localtime'))
);

-- 默认账户
INSERT OR IGNORE INTO accounts(name) VALUES ('现金'),('支付宝'),('微信'),('招行');

-- 默认支出分类（一级）
INSERT INTO categories(name,type) VALUES
  ('餐饮','expense'),('交通','expense'),('购物','expense'),
  ('娱乐','expense'),('医疗','expense'),('住房','expense'),('教育','expense');

-- 二级分类（parent_id 对应上面插入顺序）
INSERT INTO categories(parent_id,name,type) VALUES
  (1,'外卖','expense'),(1,'堂食','expense'),(1,'咖啡','expense'),(1,'零食','expense'),
  (2,'地铁','expense'),(2,'打车','expense'),(2,'公交','expense'),(2,'加油','expense'),(2,'停车','expense'),
  (3,'日用品','expense'),(3,'服饰','expense'),(3,'数码','expense'),(3,'书籍','expense'),
  (4,'游戏','expense'),(4,'电影','expense'),(4,'旅行','expense'),(4,'运动','expense'),
  (5,'挂号','expense'),(5,'药品','expense'),(5,'体检','expense'),
  (6,'房租','expense'),(6,'水电','expense'),(6,'物业','expense'),
  (7,'课程','expense'),(7,'书本','expense'),(7,'学费','expense');

-- 收入分类
INSERT INTO categories(name,type) VALUES
  ('工资','income'),('奖金','income'),('投资','income'),('兼职','income'),('报销','income');
"
```

## 记录支出

支出金额存**负数**，收入存**正数**。日期默认今天：`date('now','localtime')`。

```bash
# 花了 38 元点外卖
sqlite3 "$DB" "INSERT INTO transactions(date,amount,cat1,cat2,account,payee,note)
  VALUES(date('now','localtime'),-38,'餐饮','外卖','支付宝','美团','午饭');"

# 带三级分类
sqlite3 "$DB" "INSERT INTO transactions(date,amount,cat1,cat2,cat3,account,payee)
  VALUES('2024-03-18',-1299,'购物','数码','配件','招行','京东');"
```

## 记录收入

```bash
sqlite3 "$DB" "INSERT INTO transactions(date,amount,cat1,account,payee,note)
  VALUES(date('now','localtime'),15000,'工资','招行','公司','3月工资');"
```

## 查询：最近 N 笔

```bash
sqlite3 -column -header "$DB" \
  "SELECT date,amount,cat1,cat2,payee,note FROM transactions ORDER BY id DESC LIMIT 10;"
```

## 查询：月度支出报告

```bash
# 按一级分类汇总（当月）
sqlite3 -column -header "$DB" "
  SELECT cat1 AS 分类, printf('%.2f',SUM(ABS(amount))) AS 合计
  FROM transactions
  WHERE amount < 0 AND strftime('%Y-%m',date) = strftime('%Y-%m','now','localtime')
  GROUP BY cat1 ORDER BY 合计 DESC;"
```

## 查询：某分类明细

```bash
sqlite3 -column -header "$DB" "
  SELECT date, cat2, payee, ABS(amount) AS 金额, note
  FROM transactions
  WHERE cat1='餐饮' AND strftime('%Y-%m',date)='2024-03'
  ORDER BY date;"
```

## 查询：账户余额

```bash
sqlite3 -column -header "$DB" \
  "SELECT account AS 账户, printf('%.2f',SUM(amount)) AS 余额
   FROM transactions GROUP BY account ORDER BY account;"
```

## 查询：关键词搜索

```bash
sqlite3 -column -header "$DB" "
  SELECT date, amount, cat1, payee, note FROM transactions
  WHERE payee LIKE '%关键词%' OR note LIKE '%关键词%'
  ORDER BY date DESC LIMIT 20;"
```

## 查询：年度月度汇总

```bash
sqlite3 -column -header "$DB" "
  SELECT
    strftime('%Y-%m',date) AS 月份,
    printf('%.2f', SUM(CASE WHEN amount<0 THEN ABS(amount) ELSE 0 END)) AS 支出,
    printf('%.2f', SUM(CASE WHEN amount>0 THEN amount ELSE 0 END)) AS 收入,
    printf('%.2f', SUM(amount)) AS 结余
  FROM transactions
  WHERE strftime('%Y',date) = strftime('%Y','now','localtime')
  GROUP BY 月份 ORDER BY 月份;"
```

## 查询：预算超支检查

```bash
# 检查本月餐饮是否超过预算（如 1000 元）
BUDGET=1000
sqlite3 "$DB" "
  SELECT
    printf('%.2f', SUM(ABS(amount))) AS 已花,
    CASE WHEN SUM(ABS(amount)) > $BUDGET THEN '⚠️ 超预算' ELSE '✓ 未超' END AS 状态
  FROM transactions
  WHERE cat1='餐饮' AND amount<0
    AND strftime('%Y-%m',date)=strftime('%Y-%m','now','localtime');"
```

## 管理：查看所有分类

```bash
sqlite3 -column -header "$DB" "
  SELECT c1.name AS 一级, c2.name AS 二级, c2.type AS 类型
  FROM categories c2
  LEFT JOIN categories c1 ON c2.parent_id = c1.id
  ORDER BY c1.name, c2.name;"
```

## 管理：新增自定义分类

```bash
# 新增一级分类
sqlite3 "$DB" "INSERT INTO categories(name,type) VALUES('宠物','expense');"

# 新增二级分类（先查父级 id）
PARENT_ID=$(sqlite3 "$DB" "SELECT id FROM categories WHERE name='宠物' AND parent_id IS NULL;")
sqlite3 "$DB" "INSERT INTO categories(parent_id,name,type) VALUES($PARENT_ID,'粮食','expense');"
```

## AI 行为规范

- 首次使用检测 `$DB` 是否存在，不存在则先运行初始化 SQL
- 用户说"花了/买了/付了" → 记支出，金额取负；"收到/发工资" → 记收入，金额取正
- 省略字段时使用默认值：date=今天，account=现金，currency=CNY
- 分类不确定时根据 payee/note 推断，无法确定则用一级分类兜底（如"其他"）
- 查询结果用 `-column -header` 输出，金额保留两位小数
- 修改/删除记录时先 SELECT 确认再操作，避免误改
