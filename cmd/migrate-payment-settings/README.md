# Payment Settings Migration Tools

This directory contains migration scripts for payment settings in the PayTrack system.

## Scripts

### 1. main.go - User Payment Settings Migration
Migrates payment settings from the `users` table to the new `user_payment_methods` table.

**What it does:**
- Reads `payment_settings` from `users` table
- Creates records in `user_payment_methods` table with proper coin/network mapping
- Maps old payment types (BTC, LTC, DCR, ETH, USDT) to new format

**Usage:**
```bash
DATABASE_URL=postgres://user:pass@localhost/db go run main.go
```

### 2. migrate_payments.go - Payments Table Migration
Migrates `payment_settings` column in the `payments` table from old format to new format with network field.

**What it does:**
- Updates JSONB column `payment_settings` in `payments` table
- Transforms from old format: `{type: "btc", address: "..."}`
- To new format: `{coin: "BTC", network: "btc", address: "..."}`

**Usage:**
```bash
DATABASE_URL=postgres://user:pass@localhost/db go run migrate_payments.go
```

## Migration Mapping

| Old Type | New Coin | Network |
|----------|----------|---------|
| btc      | BTC      | btc     |
| ltc      | LTC      | ltc     |
| dcr      | DCR      | dcr     |
| eth      | ETH      | erc20   |
| usdt     | USDT     | erc20   |
| usdc     | USDC     | erc20   |

## Important Notes

1. **Run Order**: These migrations are independent and can be run in any order
2. **Backup**: Always backup your database before running migrations
3. **Idempotency**: The `migrate_payments.go` script checks if data is already in new format and skips it
4. **Transaction**: Both scripts use database transactions - if any error occurs, all changes are rolled back

## Testing

Before running on production:
1. Test on a development database copy
2. Verify the migration results
3. Check that the application works correctly with migrated data