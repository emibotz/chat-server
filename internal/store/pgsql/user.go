package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type users struct {
	pool *pgxpool.Pool
}

// 创建用户
func (s *users) Create(ctx context.Context, user *user.User) error {
	// 查询语句
	pgsql := `
INSERT INTO
	users
	(name, auth)
VALUES
	($1, $2)
RETURNING
	id
;
	`

	// 创建并返回用户的实际 ID
	row := s.pool.QueryRow(ctx, pgsql, user.Name, user.Auth)

	// 将数据库自动生成的 ID 存回用户实例中
	if err := row.Scan(&user.ID); err != nil {
		return fmt.Errorf("pgsql user create failed: %w", err)
	}

	return nil
}

// 通过 ID 获取用户，没有指定用户时返回 (nil, nil)
func (s *users) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	// 查询语句
	pgsql := `
SELECT
	id, name, auth
FROM
	users
WHERE
	id = $1
;
	`

	// 查询数据库
	row := s.pool.QueryRow(ctx, pgsql, id)

	// 扫描结果到实例中
	var u user.User
	if err := row.Scan(
		&u.ID, &u.Name, &u.Auth,
	); err != nil {

		// 如果没有用户，返回 (nil, nil)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		// 返回错误
		return nil, fmt.Errorf("pgsql user get by id failed: %w", err)
	}

	// 返回实例
	return &u, nil
}

func (s *users) GetByName(ctx context.Context, username string) (*user.User, error) {
	// 查询语句
	pgsql := `
SELECT
	id, name, auth
FROM
	users
WHERE
	name = $1
;
	`

	// 查询数据库
	row := s.pool.QueryRow(ctx, pgsql, username)

	// 扫描结果到实例中
	var u user.User
	if err := row.Scan(
		&u.ID, &u.Name, &u.Auth,
	); err != nil {

		// 如果没有用户，返回 (nil, nil)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		// 返回错误
		return nil, err
	}

	// 返回实例
	return &u, nil
}

func (s *users) Update(ctx context.Context, user *user.User) error {
	// 开启事务并自动回滚
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgsql tx begin failed: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			fmt.Printf("pgsql tx rollback failed: %v\n", err)
		}
	}()

	// 查询语句
	pgsql := `
UPDATE
	users
SET
	name=$2, auth=$3
WHERE
	id=$1
;
	`

	// 执行更新
	cmd, err := s.pool.Exec(ctx, pgsql, user.ID, user.Name, user.Auth)
	if err != nil {
		return fmt.Errorf("pgsql users update failed: %v", err)
	}

	// 如果影响的行数不等于 1 ，返回错误
	if cmd.RowsAffected() != 1 {
		return fmt.Errorf("expected 1 affected rows, got %d", cmd.RowsAffected())
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("pgsql tx commit failed: %w", err)
	}

	return nil
}

func (s *users) Delete(ctx context.Context, user *user.User) error {
	// 开启事务并自动回滚
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgsql tx begin failed: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			fmt.Printf("pgsql tx rollback failed: %v\n", err)
		}
	}()

	// 语句
	pgsql := `
DELETE FROM
	users
WHERE
	id=$1
;
	`

	// 执行删除操作
	cmd, err := s.pool.Exec(ctx, pgsql, user.ID)
	if err != nil {
		return fmt.Errorf("pgsql user delete failed: %w", err)
	}

	if cmd.RowsAffected() != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", cmd.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("pgsql tx commit failed: %w", err)
	}

	return nil
}
