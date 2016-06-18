CREATE TABLE IF NOT EXISTS `users` (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `login` varchar(255) NOT NULL UNIQUE,
  `password_hash` varchar(255) NOT NULL,
  `salt` varchar(255) NOT NULL
) DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `login_log` (
  `id` bigint NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `created_at` datetime NOT NULL,
  `user_id` int,
  `login` varchar(255) NOT NULL,
  `ip` varchar(255) NOT NULL,
  `succeeded` tinyint NOT NULL,
  INDEX idx_user_id_and_id(user_id, id),
  INDEX idx_user_id_and_succeeded(user_id, succeeded),
  INDEX idx_user_ip_and_succeeded(ip, succeeded),
  INDEX idx_user_ip_and_id(ip, id)
  -- INDEX idx_id_user_id_succeeded(user_id, succeeded, id)
) DEFAULT CHARSET=utf8;
