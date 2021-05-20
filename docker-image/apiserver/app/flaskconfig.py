"""
Flask configuration.
"""
import logging

class Config():
    DEBUG = True
    TESTING = True
    ENV = "unset"
    URL_PREFIX = '/api/v1/'
    LOG_STORE_FOLDER = '/tmp/logs'
    LOGGING_LEVEL = logging.DEBUG
    MYSQL_DATABASE_USER = "root"
    MYSQL_DATABASE_PASSWORD = "test"
    MYSQL_DATABASE_HOST = "db"
    MYSQL_DATABASE_DB = "dev"
    MYSQL_DATABASE_TABLE = "jobs"


class ProductionConfig(Config):
    ENV = "production"
