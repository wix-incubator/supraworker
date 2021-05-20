# -*- coding: utf-8 -*-
"""
Main file
"""
import os
import logging
from flask import Flask
from flask_restx.apidoc import apidoc
from app.resources.logs import logs_page
from app.resources.jobs import job_page

logging.basicConfig(
        level= logging.DEBUG,
        format='%(asctime)s {%(filename)s:%(lineno)d} %(levelname)s: %(message)s')
logging.info("Starting Flask...")
logging.info(f"Load FLASK config {os.getenv('APP_SETTINGS', 'flaskconfig.ProductionConfig')}")

app = Flask(__name__)
app.config.from_object(os.getenv('APP_SETTINGS', 'flaskconfig.ProductionConfig'))

logging.info("Register Blueprint")


apidoc.static_url_path = "{}/swagger/ui".format(app.config['URL_PREFIX'])

app.register_blueprint(job_page, url_prefix="{}/jobs".format(app.config['URL_PREFIX']))
app.register_blueprint(logs_page, url_prefix="{}/logs".format(app.config['URL_PREFIX']))

logging.info("FINISHED INITIALIZATION")
if __name__ != '__main__':
    gunicorn_logger = logging.getLogger('gunicorn.error')
    app.logger.handlers = gunicorn_logger.handlers
    app.logger.setLevel(gunicorn_logger.level)