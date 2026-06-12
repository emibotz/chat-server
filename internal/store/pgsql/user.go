package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/emibotz/chat-server/internal/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type users struct {
	pool *pgxpool.Pool
}

// 创建用户
func (s *users) Create(ctx context.Context, u *user.User) error {
	// 查询语句
	pgsql := `
INSERT INTO
	users
	(name, auth)
VALUES
	($1, $2)
ON CONFLICT
	(name)
	DO NOTHING
RETURNING
	id
;
	`

	// 创建并返回用户的实际 ID
	row := s.pool.QueryRow(ctx, pgsql, u.Name, u.Auth)

	// 将数据库自动生成的 ID 存回用户实例中
	if err := row.Scan(&u.ID); err != nil {

		// 如果数据库返回重复错误，返回重复错误
		if errors.Is(err, sql.ErrNoRows) {
			return user.ErrUserDuplicated
		}

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

// 通过多个用户 ID 查询对应用户，返回数量可能和传入数量不相同。
// [FIXME] 可能需要额外的错误处理！
func (s *users) GetByIDs(ctx context.Context, ids ...uuid.UUID) ([]*user.User, error) {
	// 带占位符的查询语句
	pgsql := `
SELECT
	id, name, auth
FROM
	users
WHERE
	id IN (%s)
	`

	// 动态生成占位符字符串，并且构建参数列表
	placeholderList := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholderList = append(placeholderList, fmt.Sprintf("$%d", i+1))
		args = append(args, id)
	}
	placeholders := strings.Join(placeholderList, ", ")

	// 构建最终查询语句
	pgsql = fmt.Sprintf(pgsql, placeholders)

	// 查询用户记录
	rows, err := s.pool.Query(ctx, pgsql, args)
	if err != nil {
		return nil, fmt.Errorf("pgsql user get by ids failed: %w", err)
	}
	defer rows.Close()

	// 扫描用户记录到列表中
	result := make([]*user.User, 0)
	for rows.Next() {
		var user user.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Auth); err != nil {
			return nil, fmt.Errorf("scan users failed: %w", err)
		}
		result = append(result, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan users failed: %w", err)
	}

	// 返回结果
	return result, nil
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
	// 开启事务并自动回滚，防止误更新多个记录
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgsql tx begin failed: %w", err)
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
	cmd, err := tx.Exec(ctx, pgsql, user.ID, user.Name, user.Auth)
	if err != nil {
		return fmt.Errorf("pgsql users update failed: %w", err)
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
	// 开启事务并自动回滚，防止误更新多个记录
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
	cmd, err := tx.Exec(ctx, pgsql, user.ID)
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
