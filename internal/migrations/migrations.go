package migrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Migration struct {
	Version int64
	Name    string
	Up      func(*gorm.DB) error
	Down    func(*gorm.DB) error
}

var registry = []Migration{
	{Version: 1, Name: "create collection tracking schema", Up: up0001, Down: down0001},
	{Version: 2, Name: "add company account rowversion", Up: up0002, Down: down0002},
	{Version: 3, Name: "enforce non-cancelled PO uniqueness", Up: up0003, Down: down0003},
}

func Status(ctx context.Context, db *gorm.DB) error {
	return withMigrationLock(ctx, db, func(tx *gorm.DB) error {
		if err := ensureRegistry(tx); err != nil {
			return err
		}
		var applied []migrationRow
		if err := tx.Table("dbo.schema_migrations").Order("version").Find(&applied).Error; err != nil {
			return fmt.Errorf("read migration status: %w", err)
		}
		for _, migration := range registry {
			found := "pending"
			for _, row := range applied {
				if row.Version == migration.Version {
					found = "applied"
				}
			}
			fmt.Printf("%04d %-40s %s\n", migration.Version, migration.Name, found)
		}
		return nil
	})
}

func Up(ctx context.Context, db *gorm.DB) error {
	return withMigrationLock(ctx, db, func(tx *gorm.DB) error {
		if err := ensureRegistry(tx); err != nil {
			return err
		}
		for _, migration := range registry {
			applied, err := isApplied(tx, migration.Version)
			if err != nil {
				return err
			}
			if applied {
				continue
			}
			if err := applyMigration(tx, migration); err != nil {
				return err
			}
		}
		return nil
	})
}

func Down(ctx context.Context, db *gorm.DB, steps int) error {
	if steps < 1 {
		return fmt.Errorf("down steps must be at least 1")
	}
	return withMigrationLock(ctx, db, func(tx *gorm.DB) error {
		if err := ensureRegistry(tx); err != nil {
			return err
		}
		var applied []migrationRow
		if err := tx.Table("dbo.schema_migrations").Order("version desc").Limit(steps).Find(&applied).Error; err != nil {
			return fmt.Errorf("read applied migrations: %w", err)
		}
		for _, row := range applied {
			migration, ok := find(row.Version)
			if !ok {
				return fmt.Errorf("migration %d is applied but not present in registry", row.Version)
			}
			if err := reverseMigration(tx, migration); err != nil {
				return err
			}
		}
		return nil
	})
}

type migrationRow struct {
	Version      int64       `gorm:"column:version"`
	Name         string      `gorm:"column:name"`
	AppliedAtUTC interface{} `gorm:"column:applied_at_utc"`
	Success      bool        `gorm:"column:success"`
}

func ensureRegistry(db *gorm.DB) error {
	return db.Exec(`
IF OBJECT_ID(N'dbo.schema_migrations', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.schema_migrations (
        version BIGINT NOT NULL CONSTRAINT PK_schema_migrations PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        applied_at_utc DATETIME2(0) NOT NULL,
        success BIT NOT NULL CONSTRAINT DF_schema_migrations_success DEFAULT (1)
    );
END`).Error
}

func isApplied(db *gorm.DB, version int64) (bool, error) {
	var count int64
	if err := db.Table("dbo.schema_migrations").Where("version = ? AND success = 1", version).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check migration %d: %w", version, err)
	}
	return count == 1, nil
}

func applyMigration(tx *gorm.DB, migration Migration) error {
	if err := migration.Up(tx); err != nil {
		return fmt.Errorf("apply migration %d: %w", migration.Version, err)
	}
	err := tx.Table("dbo.schema_migrations").Create(map[string]interface{}{
		"version":        migration.Version,
		"name":           migration.Name,
		"applied_at_utc": gorm.Expr("SYSUTCDATETIME()"),
		"success":        true,
	}).Error
	if err != nil {
		return fmt.Errorf("migration %d failed and was rolled back: %w", migration.Version, err)
	}
	fmt.Printf("applied %04d %s\n", migration.Version, migration.Name)
	return nil
}

func reverseMigration(tx *gorm.DB, migration Migration) error {
	if err := migration.Down(tx); err != nil {
		return fmt.Errorf("reverse migration %d: %w", migration.Version, err)
	}
	err := tx.Table("dbo.schema_migrations").Where("version = ?", migration.Version).Delete(&migrationRow{}).Error
	if err != nil {
		return fmt.Errorf("migration %d down failed and was rolled back: %w", migration.Version, err)
	}
	fmt.Printf("reverted %04d %s\n", migration.Version, migration.Name)
	return nil
}

func withMigrationLock(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET XACT_ABORT ON").Error; err != nil {
			return err
		}
		if err := tx.Exec(`DECLARE @result INT; EXEC @result = sp_getapplock @Resource = 'syncline-collection-tracking-migrations', @LockMode = 'Exclusive', @LockOwner = 'Transaction', @LockTimeout = 30000; IF @result < 0 THROW 51000, 'Could not acquire migration lock', 1;`).Error; err != nil {
			return fmt.Errorf("acquire migration lock: %w", err)
		}
		return fn(tx)
	})
	if err != nil {
		return fmt.Errorf("migration transaction failed: %w", err)
	}
	return nil
}

func find(version int64) (Migration, bool) {
	for _, migration := range registry {
		if migration.Version == version {
			return migration, true
		}
	}
	return Migration{}, false
}
