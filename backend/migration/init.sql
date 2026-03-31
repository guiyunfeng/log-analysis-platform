-- Log Analysis Platform Database Initialization
-- Database: log_analysis

CREATE DATABASE IF NOT EXISTS log_analysis CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE log_analysis;

CREATE TABLE IF NOT EXISTS alert_rules (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL COMMENT '规则名称',
    severity ENUM('critical', 'warning', 'noise') NOT NULL DEFAULT 'warning' COMMENT '告警级别',
    project VARCHAR(255) DEFAULT '' COMMENT '匹配项目（空=所有）',
    service VARCHAR(255) DEFAULT '' COMMENT '匹配服务（空=所有）',
    caller_file VARCHAR(255) DEFAULT '' COMMENT '匹配调用点（空=所有）',
    content_pattern VARCHAR(500) DEFAULT '' COMMENT '内容匹配（关键字或正则）',
    time_window INT NOT NULL DEFAULT 300 COMMENT '时间窗口（秒）',
    threshold INT NOT NULL DEFAULT 1 COMMENT '次数阈值',
    silence_minutes INT NOT NULL DEFAULT 30 COMMENT '静默时间（分钟）',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    rule_id BIGINT DEFAULT NULL COMMENT '触发的规则ID',
    severity ENUM('critical', 'warning', 'noise') NOT NULL COMMENT '告警级别',
    project VARCHAR(255) NOT NULL DEFAULT '',
    service VARCHAR(255) NOT NULL DEFAULT '',
    caller_file VARCHAR(255) DEFAULT '' COMMENT '调用点',
    job VARCHAR(255) DEFAULT '' COMMENT '机器标识',
    error_count INT NOT NULL DEFAULT 0 COMMENT '错误次数',
    sample_content TEXT COMMENT '示例报错内容',
    comparison VARCHAR(100) DEFAULT '' COMMENT '环比信息（如 ↑340%）',
    resolved TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已处理',
    resolved_at DATETIME DEFAULT NULL,
    notified TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已推送钉钉',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_severity (severity),
    INDEX idx_service (service),
    INDEX idx_created_at (created_at),
    INDEX idx_resolved (resolved)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS settings (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    `key` VARCHAR(255) NOT NULL COMMENT '配置键',
    value TEXT NOT NULL COMMENT '配置值',
    description VARCHAR(500) DEFAULT '' COMMENT '说明',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_key (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Preset alert rules
INSERT IGNORE INTO alert_rules (name, severity, content_pattern, time_window, threshold, silence_minutes, enabled) VALUES
('DB扫描错误', 'critical', 'Scan error on column index', 300, 0, 30, 1),
('连接失败', 'critical', 'connect failed|connection refused', 300, 0, 30, 1),
('用户未找到', 'warning', 'not found', 300, 50, 30, 1),
('风控报单错误', 'warning', 'ErrCode:205010', 300, 20, 30, 1),
('未授权请求噪音', 'noise', 'no token present in request', 300, 1, 60, 1),
('扫描器噪音', 'noise', 'CensysInspect', 300, 1, 60, 1);

-- Default settings
INSERT IGNORE INTO settings (`key`, value, description) VALUES
('spike_multiplier', '10', '突增倍数阈值'),
('global_threshold', '100', '全局默认5分钟错误阈值'),
('global_time_window', '300', '全局默认时间窗口（秒）'),
('global_silence_minutes', '30', '全局静默时间（分钟）'),
('warning_batch_interval', '5', 'warning聚合推送间隔（分钟）'),
('loki_url', '', 'Loki服务地址'),
('dingtalk_webhook', '', '钉钉机器人Webhook地址');
