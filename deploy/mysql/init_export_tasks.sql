-- 创建导出任务表 export_tasks，用于异步导出队列
CREATE TABLE IF NOT EXISTS export_tasks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  requested_by BIGINT UNSIGNED NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  start_time DATETIME NULL,
  end_time DATETIME NULL,
  statuses VARCHAR(255) DEFAULT NULL,
  file_path VARCHAR(1024) DEFAULT NULL,
  file_name VARCHAR(255) DEFAULT NULL,
  error_msg TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_status (status),
  INDEX idx_requested_by (requested_by),
  INDEX idx_created_at (created_at)
);
