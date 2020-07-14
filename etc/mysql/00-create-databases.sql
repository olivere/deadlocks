-- deadlocks
CREATE DATABASE IF NOT EXISTS `deadlocks`;
CREATE USER IF NOT EXISTS 'go' IDENTIFIED BY 'go';
GRANT ALL ON `deadlocks`.* TO 'go';
