-- MySQL dump 10.13  Distrib 8.0.19, for osx10.15 (x86_64)
--
-- Host: 127.0.0.1    Database: crypto
-- ------------------------------------------------------
-- Server version	5.5.5-10.6.2-MariaDB-1:10.6.2+maria~focal

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `contract_strategies`
--

DROP TABLE IF EXISTS `contract_strategies`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `contract_strategies` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'AI_id',
  `uuid` char(36) NOT NULL COMMENT 'uuid',
  `user_uuid` char(36) NOT NULL COMMENT 'User uuid',
  `symbol` varchar(20) NOT NULL COMMENT 'Symbol e.g. BTC-PERP',
  `cost` decimal(18,8) unsigned NOT NULL COMMENT 'Cost',
  `contract_params` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'Contract params',
  `enabled` tinyint(3) unsigned NOT NULL DEFAULT 0 COMMENT '1: enabled 0: disabled',
  `position_type` tinyint(4) unsigned NOT NULL COMMENT '1: long 0: short',
  `position_status` tinyint(4) unsigned NOT NULL DEFAULT 0 COMMENT '2: unknown 1: opened 0: closed',
  `exchange` varchar(20) NOT NULL COMMENT 'Exchange name e.g. FTX',
  `exchange_orders_details` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'Bespoke orders details by exchange',
  `last_position_at` datetime DEFAULT NULL COMMENT 'Last position created time',
  `created_at` datetime NOT NULL DEFAULT current_timestamp() COMMENT 'Create time',
  `updated_at` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp() COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uuid` (`uuid`),
  KEY `enabled_positionStatus` (`enabled`,`position_status`),
  KEY `updated_at` (`updated_at`),
  KEY `userUuid_positionStatus_enabled` (`user_uuid`,`position_status`,`enabled`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Contract Strategies';
/*!40101 SET character_set_client = @saved_cs_client */;
