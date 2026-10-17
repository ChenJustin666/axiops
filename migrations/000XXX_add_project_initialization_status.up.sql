-- 添加项目初始化状态字段
ALTER TABLE project ADD COLUMN initialization_status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT '初始化状态: pending(待初始化), in_progress(初始化中), completed(已完成)';
ALTER TABLE project ADD COLUMN initialization_step INT NOT NULL DEFAULT 0 COMMENT '当前初始化步骤: 0-未开始, 1-项目配置, 2-服务配置, 3-环境配置, 4-工作流配置';
ALTER TABLE project ADD COLUMN initialized_at DATETIME NULL COMMENT '完成初始化时间';

-- 为已存在的项目设置为已完成状态
UPDATE project SET initialization_status = 'completed', initialization_step = 4, initialized_at = updated_at WHERE id > 0;
