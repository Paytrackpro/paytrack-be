# 🚀 MGMT-NG Backend (`mgmtngd`)

> Backend service for the MGMT-NG project. Built with [Go](https://golang.org/) (version 1.19).

---

## 📘 Application Scope

See full documentation in the [Project Wiki](https://code.cryptopower.dev/mgmt-ng/fe/-/wikis/home)

---

## 🗄️ Setup

### 1. Install Go

Ensure you have Go 1.19 installed: [https://go.dev/dl/](https://go.dev/dl/)

### 2. Setup PostgreSQL

Install [PostgreSQL](https://www.postgresql.org/) and create a database named `mgmtng`:

```sql
CREATE DATABASE mgmtng;
```

---

## ⚙️ Configuration

Create a config file at `./private/config.yaml`. You can copy from the sample:

```bash
cp sample/mgmtngd.yaml private/config.yaml
```

Edit the `db` section to match your environment:

```yaml
db:
  dns: "host=<host> user=<user> password=<password> dbname=mgmtng port=<port> sslmode=disable TimeZone=Asia/Shanghai"
```

---

## ▶️ Running `mgmtngd`

### Option 1: Run from terminal

```bash
go run ./cmd/mgmtngd --config=./private/config.yaml
```

### Option 2: Using Makefile

Create a `Makefile` with the following content:

```makefile
.PHONY: up

up:
	go run ./cmd/mgmtngd --config=./private/config.yaml
```

Then run:

```bash
make up
```

---

## 👤 Create Admin User

Access the database and run the following SQL to promote a user to admin:

```sql
UPDATE users SET role = 1 WHERE user_name = '<username>';
```

Example:

```sql
UPDATE users SET role = 1 WHERE user_name = 'justindo';
```

---

## 🧾 License

This project is licensed under the **MIT License**.
