package department

import (
	"context"
	"database/sql"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context) ([]Department, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT d.id,d.name,d.parent_id,d.sort_order,d.leader_user_id,
		(SELECT COUNT(*) FROM user u WHERE u.department_id=d.id AND u.deleted_at IS NULL),
		d.created_by,d.created_at,d.updated_at
		FROM department d ORDER BY d.sort_order,d.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Department, 0)
	for rows.Next() {
		var item Department
		if err := rows.Scan(&item.ID, &item.Name, &item.ParentID, &item.SortOrder, &item.LeaderUserID,
			&item.UserCount, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Exists(ctx context.Context, id int64) (bool, error) {
	if id == 0 {
		return true, nil
	}
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM department WHERE id=?", id).Scan(&count)
	return count > 0, err
}

func (r *Repository) Create(ctx context.Context, item *Department) (int64, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO department(name,parent_id,sort_order,leader_user_id,created_by)
		VALUES(?,?,?,?,?)`, item.Name, item.ParentID, item.SortOrder, item.LeaderUserID, item.CreatedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) Update(ctx context.Context, item *Department) error {
	_, err := r.db.ExecContext(ctx, `UPDATE department SET name=?,parent_id=?,sort_order=?,leader_user_id=? WHERE id=?`,
		item.Name, item.ParentID, item.SortOrder, item.LeaderUserID, item.ID)
	return err
}

func (r *Repository) IsDescendant(ctx context.Context, ancestorID, candidateID int64) (bool, error) {
	if candidateID == 0 {
		return false, nil
	}
	current := candidateID
	for current > 0 {
		if current == ancestorID {
			return true, nil
		}
		var parentID int64
		err := r.db.QueryRowContext(ctx, "SELECT parent_id FROM department WHERE id=?", current).Scan(&parentID)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		current = parentID
	}
	return false, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) (bool, error) {
	var references int
	if err := r.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM department WHERE parent_id=?) +
		(SELECT COUNT(*) FROM user WHERE department_id=? AND deleted_at IS NULL)`, id, id).Scan(&references); err != nil {
		return false, err
	}
	if references > 0 {
		return false, nil
	}
	result, err := r.db.ExecContext(ctx, "DELETE FROM department WHERE id=?", id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}
