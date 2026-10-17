-- 回滚：删除项目初始化状态字段
ALTER TABLE project DROP COLUMN initialization_status;
ALTER TABLE project DROP COLUMN initialization_step;
ALTER TABLE project DROP COLUMN initialized_at;
