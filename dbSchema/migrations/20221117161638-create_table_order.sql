
-- +migrate Up
CREATE TABLE IF NOT EXISTS `be-match`.`match_record`
(
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'id',
    `order_id` BIGINT UNSIGNED NOT NULL COMMENT '訂單id',
    `member_id` BIGINT UNSIGNED NOT NULL COMMENT '會員id',
    `position_id` BIGINT UNSIGNED NULL DEFAULT NULL COMMENT '倉位id',
    `match_status` TINYINT(4) NOT NULL COMMENT '訂單狀態 1:待處理 2:失敗 3:完成 4:取消 5:回滾',
    `transaction_type` TINYINT(4) NOT NULL COMMENT '交易類別 1:開倉 2:關倉',
    `exchange_code` VARCHAR(32) NOT NULL COMMENT '交易所代號',
    `product_code` VARCHAR(32) NOT NULL COMMENT '產品代號',
    `trade_type` TINYINT(4) NOT NULL COMMENT '買賣類別 1:買 2:賣',
    `open_price` DECIMAL(19,4) NULL DEFAULT NULL COMMENT '開倉價',
    `close_price` DECIMAL(19,4) NULL DEFAULT NULL COMMENT '關倉價',
    `amount` DECIMAL(19,4) NOT NULL COMMENT '交易數量, 開倉時代表開多少倉, 關倉時代表關多少倉',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '創建時間',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新時間',

    PRIMARY KEY (`id`),
    UNIQUE INDEX (`member_id`,`exchange_code`, `product_code`,`created_at`)
) AUTO_INCREMENT=1 CHARSET=`utf8mb4` COLLATE=`utf8mb4_general_ci` COMMENT '撮合紀錄';


-- +migrate Down
SET FOREIGN_KEY_CHECKS=0;
DROP TABLE IF EXISTS `match_record`;